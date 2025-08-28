// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package controller

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	dmscrypto "gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/sys"
)

var log = logging.Logger("actor")

var _ GlusterControllerInterface = (*GlusterController)(nil)

// GlusterControllerInterface defines the contract for a GlusterFS controller.
type GlusterControllerInterface interface {
	CreateVolume(volName, clientPEM string) (string, error)
	StartVolume(volName string) error
	DeleteVolume(volName string) error
	CheckServer() error
	IsServerWorking() bool
}

const fuseModel = "fuse"

// GlusterController is responsible for managing GlusterFS volumes.
type GlusterController struct {
	glusterfsServerHostname string
	bricksDir               string
	caAuthority             string
}

// NewGlusterController creates a new instance of GlusterController.
func NewGlusterController(glusterfsServerHostname, bricksDir, caDir string) (*GlusterController, error) {
	if caDir == "" {
		return nil, errors.New("glusterfs CA directory is empty")
	}

	if !isModuleLoaded(fuseModel) {
		err := loadModule(fuseModel)
		if err != nil {
			log.Warnf("failed to load fuse kernel module: %v", err)
		}
	}

	g := &GlusterController{
		glusterfsServerHostname: glusterfsServerHostname,
		bricksDir:               bricksDir,
		caAuthority:             caDir,
	}

	g.ensureDirectories()

	// check if the glusterfs_nodes contains a list of server certificates
	empty, err := isPEMDirectoryEmpty(filepath.Join(g.caAuthority, "glusterfs_nodes"))
	if err != nil {
		return nil, fmt.Errorf("failed to check glusterfs_nodes for server certificates: %w", err)
	}

	if empty {
		return nil, errors.New("glusterfs_nodes must contain server certificates")
	}

	return g, nil
}

func (gc *GlusterController) ensureDirectories() {
	dirs := []string{
		filepath.Join(gc.caAuthority, "glusterfs_nodes"),
		filepath.Join(gc.caAuthority, "clients"),
		gc.bricksDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
}

func (gc *GlusterController) createCACerts(folders []string, output string) error {
	caFilePath := filepath.Join(gc.caAuthority, output)

	var caContent strings.Builder

	for _, folder := range folders {
		files, err := os.ReadDir(folder)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", folder, err)
		}

		for _, file := range files {
			if filepath.Ext(file.Name()) == ".pem" {
				filePath := filepath.Join(folder, file.Name())
				content, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read file %s: %w", filePath, err)
				}
				caContent.Write(content)
			}
		}
	}

	if err := os.WriteFile(caFilePath, []byte(caContent.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write CA file %s: %w", caFilePath, err)
	}

	return nil
}

// generateGlusterFSServerCA concatenates all .pem files in glusterfs_nodes and clients folders into a single file "glusterfs.ca"
func (gc *GlusterController) generateGlusterFSServerCA() error {
	folders := []string{
		filepath.Join(gc.caAuthority, "glusterfs_nodes"),
		filepath.Join(gc.caAuthority, "clients"),
	}
	return gc.createCACerts(folders, "glusterfs.ca")
}

// generateGlusterFSClientCA concatenates all .pem files in glusterfs_nodes folders into a single file "glusterfs-client.ca"
func (gc *GlusterController) generateGlusterFSClientCA() error {
	folders := []string{
		filepath.Join(gc.caAuthority, "glusterfs_nodes"),
	}
	return gc.createCACerts(folders, "glusterfs-client.ca")
}

// enableTLS enables tls for the volume.
func (gc *GlusterController) enableTLS(volName string) error {
	cmds := [][]string{
		{"volume", "set", volName, "server.ssl", "on"},
		{"volume", "set", volName, "client.ssl", "on"},
	}

	for _, cmd := range cmds {
		output, err := sys.ExecCommand("gluster", cmd...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to set TLS option %v for volume %s: %v, output: %s", cmd, volName, err, string(output))
		}
	}
	return nil
}

// CreateVolume creates a new GlusterFS volume.
func (gc *GlusterController) CreateVolume(volName string, clientPem string) (string, error) {
	err := validatePEM([]byte(clientPem))
	if err != nil {
		return "", fmt.Errorf("failed to validate pem: %w", err)
	}

	_, err = gc.saveHashedContent(clientPem)
	if err != nil {
		return "", fmt.Errorf("failed to save client certificate: %w", err)
	}

	// create a random brick name
	randomBytes, err := dmscrypto.RandomEntropy(20)
	if err != nil {
		return "", fmt.Errorf("failed to create random brick name: %w", err)
	}

	generatedBrickName, err := dmscrypto.Sha3(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate brick hash: %w", err)
	}

	args := []string{"volume", "create", volName, fmt.Sprintf("%s:%s", gc.glusterfsServerHostname, filepath.Join(gc.bricksDir, hex.EncodeToString(generatedBrickName)))}

	// force create
	args = append(args, "force")

	output, err := sys.ExecCommand("gluster", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create volume %s: %v, output: %s", volName, err, string(output))
	}

	// refresh the CA
	if err := gc.generateGlusterFSServerCA(); err != nil {
		return "", fmt.Errorf("failed to reload glusterfs server ca: %w", err)
	}

	// refresh client CA
	if err := gc.generateGlusterFSClientCA(); err != nil {
		return "", fmt.Errorf("failed to reload glusterfs client ca: %w", err)
	}

	if err := gc.enableTLS(volName); err != nil {
		return "", fmt.Errorf("failed to enable TLS for volume %s: %v", volName, err)
	}

	clientCert, err := os.ReadFile(filepath.Join(gc.caAuthority, "glusterfs-client.ca"))
	if err != nil {
		return "", fmt.Errorf("failed to read the client ca file: %w", err)
	}

	return string(clientCert), nil
}

// StartVolume starts a given GlusterFS volume.
func (gc *GlusterController) StartVolume(volName string) error {
	output, err := sys.ExecCommand("gluster", "volume", "start", volName, "force").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start volume %s: %v, output: %s", volName, err, string(output))
	}
	return nil
}

// DeleteVolume stops and deletes the specified GlusterFS volume.
func (gc *GlusterController) DeleteVolume(volName string) error {
	output, err := sys.ExecCommand("gluster", "volume", "stop", volName, "--mode=script").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop volume %s: %v, output: %s", volName, err, string(output))
	}
	// Delete the volume
	output, err = sys.ExecCommand("gluster", "volume", "delete", volName, "--mode=script").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete volume %s: %v, output: %s", volName, err, string(output))
	}
	return nil
}

// CheckServer executes a simple GlusterFS command ("gluster pool list") to verify
// that the glusterd daemon is running and responsive.
func (gc *GlusterController) CheckServer() error {
	output, err := sys.ExecCommand("gluster", "pool", "list").CombinedOutput()
	if err != nil {
		return fmt.Errorf("glusterfs server check failed: %w, output: %s", err, output)
	}
	return nil
}

// IsServerWorking returns true if CheckServer does not report any errors.
func (gc *GlusterController) IsServerWorking() bool {
	return gc.CheckServer() == nil
}

func (gc *GlusterController) saveHashedContent(content string) (string, error) {
	hash := sha256.Sum256([]byte(content))
	hashString := hex.EncodeToString(hash[:])
	filePath := filepath.Join(gc.caAuthority, "clients", hashString+".pem")

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return filePath, nil
}

// isModuleLoaded checks if a given kernel module is loaded
func isModuleLoaded(module string) bool {
	output, err := sys.ExecCommand("lsmod").CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), module)
}

func loadModule(module string) error {
	_, err := sys.ExecCommand("modprobe", module).CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}

// validatePEM checks if a given file is a valid PEM-encoded certificate or key
func validatePEM(data []byte) error {
	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return errors.New("invalid PEM")
		}

		switch block.Type {
		case "CERTIFICATE":
			if _, err := x509.ParseCertificate(block.Bytes); err != nil {
				return fmt.Errorf("invalid certificate: %w", err)
			}
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
				if _, errRSA := x509.ParsePKCS1PrivateKey(block.Bytes); errRSA != nil {
					if _, errEC := x509.ParseECPrivateKey(block.Bytes); errEC != nil {
						return errors.New("invalid certificate")
					}
				}
			}
		}
	}

	return nil
}

func isPEMDirectoryEmpty(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".pem" {
			return false, nil
		}
	}
	return true, nil
}
