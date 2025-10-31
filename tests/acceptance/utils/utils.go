// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	dutils "gitlab.com/nunet/device-management-service/utils"

	"gitlab.com/nunet/device-management-service/dms/node"
)

func MultiaddrFromCLI(info *node.PeerAddrInfoResponse) (string, error) {
	addrs := strings.Split(info.Address, ", ")
	var addr string
	for _, a := range addrs {
		maString := a + "/p2p/" + info.ID
		ma, err := multiaddr.NewMultiaddr(maString)
		if err != nil {
			return "", err
		}

		if manet.IsIPLoopback(ma) {
			continue
		}
		addr = ma.String()
		break
	}
	return addr, nil
}

func CreateContext(i *Instance, name string) (*Context, error) {
	name = strings.ToLower(name)

	did, err := i.RunDMSCmd(fmt.Sprintf("nunet key new %s", name))
	if err != nil {
		return nil, err
	}
	did = strings.TrimSpace(did)

	_, err = i.RunDMSCmd(fmt.Sprintf("nunet cap new %s", name))
	if err != nil {
		return nil, err
	}

	context := &Context{
		Name:     name,
		DID:      did,
		instance: i,
	}

	i.Contexts[name] = context

	return context, nil
}

// JoinOrg allows a node to join an existing organization
func JoinOrg(user, dms *Context, orgDID, grantFromOrg string) error {
	err := user.Anchor("provide", grantFromOrg)
	if err != nil {
		return fmt.Errorf("could not anchor cap: %w", err)
	}

	grantToken, err := user.Grant(orgDID)
	if err != nil {
		return fmt.Errorf("failed to grant: %w", err)
	}

	err = dms.Anchor("require", grantToken)
	if err != nil {
		return err
	}

	delegateToken, err := user.Delegate(dms.DID)
	if err != nil {
		return fmt.Errorf("failed to delegate: %w", err)
	}

	err = dms.Anchor("provide", delegateToken)
	if err != nil {
		return err
	}

	return nil
}

func SetupPrivateNetwork(user, dms, org *Context) error {
	_, err := user.Grant(org.DID)
	if err != nil {
		return err
	}

	orgGrant, err := org.Grant(user.DID)
	if err != nil {
		return err
	}

	err = JoinOrg(user, dms, org.DID, orgGrant)
	if err != nil {
		return err
	}
	return nil
}

func FindTestdata(name string) string {
	here := dutils.CurrentFileDirectory()
	return filepath.Join(here, "..", "tests", "acceptance", "testdata", name)
}

func UploadFile(i *Instance, source string) (dest string, err error) {
	file := filepath.Base(source)
	dest = filepath.Join("/root", file)
	err = i.UploadFile(source, dest, 0o755)
	if err != nil {
		return "", err
	}
	return dest, nil
}

// NodeWithDMS retrieves a node from the node map
func NodeWithDMS(nodeMap map[string]*Node, nodeName string) (*Instance, *Context) {
	nodeName = strings.ToLower(nodeName)
	node, ok := nodeMap[nodeName]
	if !ok {
		return nil, nil
	}

	return node.Instance, node.DMS()
}

func UploadScripts(i *Instance, ensemble string) (err error) {
	// Upload scripts listed in the ensemble file if needed
	output, err := i.RunCMD([]string{"yq", "e", ".scripts // [] | .[]", ensemble})
	if err != nil {
		return err
	}
	out := strings.TrimSpace(output)
	for scriptName := range strings.SplitSeq(out, "\n") {
		if scriptName == "" {
			continue
		}
		scriptPath := fmt.Sprintf("scripts/%s", scriptName)
		file := FindTestdata(scriptPath)

		script, err := UploadFile(i, file)
		if err != nil {
			return err
		}
		if script == "" {
			return fmt.Errorf("script is empty")
		}
	}
	return nil
}
