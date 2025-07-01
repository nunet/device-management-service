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

func UploadEnsemble(node *Node, filename string, peer string) (file string, err error) {
	here := dutils.CurrentFileDirectory()
	source := filepath.Join(here, "..", "examples", filename)
	dest := fmt.Sprintf("/root/%s", filename)

	err = node.UploadFile(source, dest, 0o755)
	if err != nil {
		return "", err
	}

	// update the ensemble configuration to specify compute provider peer ID
	updateCmd := fmt.Sprintf("sed -i 's/failure_recovery: stay_down/failure_recovery: stay_down\\n        peer: %s/' %s",
		peer, dest)
	_, err = node.RunCMD([]string{"sh", "-c", updateCmd})
	if err != nil {
		return "", err
	}
	return dest, nil
}
