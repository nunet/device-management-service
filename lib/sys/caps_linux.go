// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
