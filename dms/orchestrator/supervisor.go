// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

func (o *BasicOrchestrator) supervise() {
	log.Debugf("Starting supervision for allocations: %+v", o.manifest.Allocations)
	expiry := uint64(time.Now().Add(5 * time.Second).UnixNano())
	wg := sync.WaitGroup{}

	for allocName, allocation := range o.manifest.Allocations {
		// skip empty healthchecks
		if o.manifest.Allocations[allocName].Healthcheck.Type == "" {
			continue
		}

		msg, err := actor.Message(
			o.actor.Handle(),
			allocation.Handle,
			behaviors.RegisterHealthcheckBehavior,
			behaviors.RegisterHealthcheckRequest{
				EnsembleID:  o.id,
				HealthCheck: o.manifest.Allocations[allocName].Healthcheck,
			},
			actor.WithMessageExpiry(expiry),
		)
		if err != nil {
			log.Errorf("failed to create supervisor message: %s", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			replyCh, err := o.actor.Invoke(msg)
			if err != nil {
				log.Errorf("failed to invoke heartbeat on allocation: %s", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response behaviors.RegisterHealthcheckResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					log.Errorf("error unmarshalling supervisor reply: %s", err)
					return
				}

				if !response.OK {
					log.Errorf("error registering healthcheck: %s", response.Error)
					return
				}
			case <-time.After(time.Second * 5):
				log.Errorf("timeout waiting for supervisor reply")
				return
			}

			log.Info("successfully registered healthcheck for allocation: %s", allocation.ID)
		}()
	}

	wg.Wait()

	ticker := time.NewTicker(actor.HealthCheckInterval)
	defer ticker.Stop()

	failures := map[string]int{}
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			expiry := uint64(time.Now().Add(5 * time.Second).UnixNano())
			wg := sync.WaitGroup{}
			for n, allocation := range o.manifest.Allocations {
				msg, err := actor.Message(
					o.actor.Handle(),
					allocation.Handle,
					actor.HealthCheckBehavior,
					struct{}{},
					actor.WithMessageExpiry(expiry),
				)
				if err != nil {
					log.Errorf("failed to create supervisor message: %s", err)
					continue
				}

				wg.Add(1)
				go func() {
					defer wg.Done()
					if o.isTerminatedTask(n) {
						return
					}

					replyCh, err := o.actor.Invoke(msg)
					if err != nil {
						log.Errorf("failed to invoke heartbeat on allocation: %s", err)
						return
					}

					ticker := time.NewTicker(time.Second * 5)
					defer ticker.Stop()

					var reply actor.Envelope
					select {
					case reply = <-replyCh:
						defer reply.Discard()

						var resp behaviors.HealthCheckResponse
						if err := json.Unmarshal(reply.Message, &resp); err != nil {
							log.Errorf("error unmarshalling supervisor reply: %s", err)
							return
						}

						if !resp.OK {
							log.Errorf("error in healthcheck: %s", resp.Error)

							o.lock.Lock()
							failures[allocation.Handle.ID.String()]++
							v := failures[allocation.Handle.ID.String()]
							o.lock.Unlock()
							if v >= 3 {
								if err := o.escalateFailure(allocation.Handle); err != nil {
									log.Errorf("failed to escalate failure: %s", err)
								} else {
									log.Debug("escalated failure, resetting healthcheck failures counter")
									delete(failures, allocation.Handle.ID.String())
								}
							}
							return
						} else {
							log.Infof("successfully healthchecked allocation %s", allocation.ID)
							delete(failures, allocation.Handle.ID.String())
							return
						}

					case <-ticker.C:
						if o.isTerminatedTask(n) {
							return
						}

						log.Warnf("timeout waiting for supervisor reply")
						o.lock.Lock()
						failures[allocation.Handle.ID.String()]++
						v := failures[allocation.Handle.ID.String()]
						o.lock.Unlock()
						if v >= 3 {
							if err := o.escalateFailure(allocation.Handle); err != nil {
								log.Errorf("failed to escalate failure: %s", err)
							} else {
								log.Debug("escalated failure, resetting healthcheck failures counter")
								delete(failures, allocation.Handle.ID.String())
							}
							return
						}
					}
				}()
			}

			wg.Wait()
		}
	}
}

func (o *BasicOrchestrator) escalateFailure(allocHandle actor.Handle) error {
	// TODO we need to decide how to handle repeated failures and also correlated failures
	//      from a node.
	//      Also, we should not restart at first failure, but wait for a number of
	//      consecutive failures.
	//      See https://gitlab.com/nunet/device-management-service/-/issues/794
	log.Debugf("escalating failure for allocation %s", allocHandle.String())
	expiry := uint64(time.Now().Add(5 * time.Second).UnixNano())
	msg, err := actor.Message(
		o.actor.Handle(),
		allocHandle,
		behaviors.AllocationRestartBehavior,
		struct{}{},
		actor.WithMessageExpiry(expiry),
	)
	if err != nil {
		return err
	}

	replyCh, err := o.actor.Invoke(msg)
	if err != nil {
		return err
	}

	select {
	case reply := <-replyCh:
		reply.Discard()
		return nil
	case <-time.After(time.Minute * 2):
		return fmt.Errorf("timeout waiting for supervisor reply")
	}
}

func (o *BasicOrchestrator) isTerminatedTask(name string) bool {
	a, ok := o.manifest.Allocations[name]
	if !ok {
		return false
	}

	if a.Type == jtypes.AllocationTypeTask &&
		a.Status != jtypes.AllocationRunning {
		return true
	}
	return false
}

func (o *BasicOrchestrator) isOnlyTaskEnsemble() bool {
	for _, a := range o.manifest.Allocations {
		if a.Type != jtypes.AllocationTypeTask {
			return false
		}
	}
	return true
}

// monitorOnlyTasksEnsemble will be responsible for tearing down
// the orchestrator after all tasks are terminated.
func (o *BasicOrchestrator) monitorOnlyTasksEnsemble() {
	if !o.isOnlyTaskEnsemble() {
		return
	}

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
selectLoop:
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			for name := range o.manifest.Allocations {
				if !o.isTerminatedTask(name) {
					continue selectLoop
				}
			}
			log.Infof("All tasks are terminated, shutting down orchestrator.")

			o.setStatus(jtypes.DeploymentStatusCompleted)
			o.cancel()
			return
		}
	}
}
