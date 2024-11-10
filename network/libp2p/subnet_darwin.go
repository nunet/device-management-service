// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build darwin

//nolint:all // This is a stub file for darwin
package libp2p

import (
	"fmt"
)

func (l *Libp2p) MapPort(subnetID, protocol, sourceIP, sourcePort, destIP, destPort string) error {
	// TODO track the port so that we can unmap it when we tear down the subnet
	return fmt.Errorf("TODO MapPort darwin")
}

func (l *Libp2p) UnmapPort(subnetID, protocol, sourceIP, sourcePort, destIP, destPort string) error {
	// TODO
	return fmt.Errorf("TODO UnmapPort darwin")
}
