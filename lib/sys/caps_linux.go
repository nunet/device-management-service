//go:build linux

package sys

import (
	"fmt"

	"kernel.org/pub/linux/libs/security/libcap/cap"
)

// RequiredCaps checks if the required capabilities are set
func RequiredCaps() error {
	caps := cap.GetProc()
	adminP, err := caps.GetFlag(cap.Permitted, cap.NET_ADMIN)
	if err != nil {
		return fmt.Errorf("error getting NET_ADMIN flag: %w", err)
	}

	adminE, err := caps.GetFlag(cap.Effective, cap.NET_ADMIN)
	if err != nil {
		return fmt.Errorf("error getting NET_ADMIN flag: %w", err)
	}

	if adminP && adminE {
		return nil
	}

	return fmt.Errorf("required capability NET_ADMIN not set")
}
