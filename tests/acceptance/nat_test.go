// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build acceptance || !unit

package acceptance

import (
	"testing"

	"github.com/cucumber/godog"
	"gitlab.com/nunet/device-management-service/tests/acceptance/steps"
)

func TestNAT(t *testing.T) {
	o := opts
	o.TestingT = t
	o.Paths = []string{"features/nat.feature"}

	suite := godog.TestSuite{
		Name:                "nat",
		Options:             &o,
		ScenarioInitializer: steps.NAT,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
