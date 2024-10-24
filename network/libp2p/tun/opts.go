// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package tun

// Option defines a TUN device modifier option.
type Option func(tun *TUN) error

// Address sets the local address and subnet for an interface.
// On MacOS devices use this function to set the Src Address
// for an interface and use DestAddress to set the destination ip.
func Address(address string) Option {
	return func(tun *TUN) error {
		return tun.setAddress(address)
	}
}

// MTU sets the Maximum Transmission Unit size for an interface.
func MTU(mtu int) Option {
	return func(tun *TUN) error {
		return tun.setMTU(mtu)
	}
}
