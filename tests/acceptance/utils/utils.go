// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
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

func SetupPrivateNetwork(user, dms, org *Context) error {
	_, err := user.Grant(org.DID)
	if err != nil {
		return err
	}

	orgGrant, err := org.Grant(user.DID)
	if err != nil {
		return err
	}

	err = user.JoinOrg(dms, org.DID, orgGrant)
	if err != nil {
		return err
	}
	return nil
}

func FindTestdata(name string) string {
	here := dutils.CurrentFileDirectory()
	return filepath.Join(here, "..", "tests", "acceptance", "testdata", name)
}

func UploadEnsemble(node *Node, source string) (dest string, err error) {
	file := filepath.Base(source)
	dest = filepath.Join("/root", file)
	err = node.UploadFile(source, dest, 0o755)
	if err != nil {
		return "", err
	}
	return dest, nil
}

// NodeWithDMS retrieves a node and its DMS context from the node map
func NodeWithDMS(nodeMap map[string]*Node, nodeName string) (*Node, *Context) {
	nodeName = strings.ToLower(nodeName)
	node, ok := nodeMap[nodeName]
	if !ok {
		return nil, nil
	}

	dmsCtx, ok := node.Contexts[nodeName+DefaultDMSSuffix]
	if !ok {
		return nil, nil
	}

	return node, dmsCtx
}
