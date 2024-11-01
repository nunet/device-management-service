// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package dms

import (
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/multierr"
)

var log = logging.Logger("dms")

// silenceLibp2pLogging is used to silence logs coming from libp2p
// imported libraries as they're enabled by default.
//
// TODO: move this to observability?
func silenceLibp2pLogging() error {
	log.Debug("silecing libp2p logging")
	var errs error

	err := logging.SetLogLevel("libp2p", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("swarm2", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("basichost", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("pubsub", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("p2p-config", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("routedhost", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("relay", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("autorelay", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("autonat", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("node", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("p2p-holepunch", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevel("rcmgr", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevelRegex("dht/*", "panic")
	errs = multierr.Append(errs, err)

	err = logging.SetLogLevelRegex("net/*", "panic")
	errs = multierr.Append(errs, err)

	return errs
}
