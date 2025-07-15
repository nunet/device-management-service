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
