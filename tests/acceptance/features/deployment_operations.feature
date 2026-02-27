Feature: Deployment Operations
  View, manage, and interact with active or failed deployments.

  Background:
    Given the following nodes
      | nodes   | role | onboarded | org   |
      | Alice   | SP   | false     | nunet |
      | Bob     | CP   | true      | nunet |

  Scenario Outline: Maintain history after restart of task
    Given "Alice" has 1 task <status> on "Bob"
    And "Alice" restarts DMS
    When "Alice" list deployments
    Then "Alice" should see the task restored

  Examples:
    |     status     |
    |   "Completed"  |
    |    "Running"   |
    |  "Committing"  |
    | "Provisioning" |

  Scenario Outline: Maintain history after restart of service
    Given "Alice" has 1 service <status> on "Bob"
    And "Alice" restarts DMS
    When "Alice" list deployments
    Then "Alice" should see the service restored

  Examples:
    |     status     |
    |    "Running"   |
    |  "Committing"  |
    | "Provisioning" |

  Scenario: Prune deployment
    Given "Alice" has 1 task completed on "Bob"
    When "Alice" prunes the deployment
    Then "Alice" should see deployment list empty

  @wip
  @complexity:low
  Scenario: List active deployments

  @wip
  @complexity:low
  Scenario: View a deployment’s current status

  @wip
  @complexity:low
  Scenario: View deployment manifest

  @wip
  @complexity:low
  Scenario: View deployment logs

  @wip
  @complexity:low
  Scenario: Restart a failed deployment

  @wip
  @complexity:low
  Scenario: Shutdown/stop a running deployment

  Scenario: List deployments with pagination and status filter
    Given "Alice" has 2 tasks with status "Running" on "Bob"
    And "Alice" has 1 task with status "Completed" on "Bob"
    When "Alice" lists deployments filtered by status "Running" with limit 1 and offset 0
    Then "Alice" should see 1 deployment
    And all deployments should have status "Running"
    And the response should indicate true more results available
    And the response should have total 2
    When "Alice" lists deployments filtered by status "Running" with limit 1 and offset 1
    Then "Alice" should see 1 deployment
    And all deployments should have status "Running"
    And the response should indicate false more results available
    And the response should have total 2

  Scenario: Get comprehensive deployment information
    Given "Alice" has 1 service "Running" on "Bob"
    When "Alice" gets deployment info for the deployment
    Then "Alice" should receive deployment info with status, manifest, and allocations
    And the deployment info should contain valid allocation details

  Scenario: Get deployment information with resource usage
    Given "Alice" has 1 service "Running" on "Bob"
    When "Alice" gets deployment info with usage for the deployment
    Then "Alice" should receive deployment info with usage statistics
    And each allocation should have executor stats in allocation details

  Scenario: Get deployment information with log paths
    Given "Alice" has 1 service "Running" on "Bob"
    When "Alice" gets deployment info with logs for the deployment
    Then "Alice" should receive deployment info with log paths
    And log paths should be valid file paths
    And log files should exist at the specified paths

  Scenario: Get deployment information with logs for specific allocations
    Given "Alice" has 1 service "Running" on "Bob"
    When "Alice" gets deployment info with logs for allocation "alloc1"
    Then "Alice" should receive deployment info with log paths only for "alloc1"

  Scenario: Get complete deployment information (all options)
    Given "Alice" has 1 service "Running" on "Bob"
    When "Alice" gets complete deployment info (with usage and logs) for the deployment
    Then "Alice" should receive deployment info with status, manifest, allocations, usage, and log paths
    And all response fields should be populated correctly

  Scenario: Get deployment information for non-running deployment
    Given "Alice" has 1 task "Completed" on "Bob"
    When "Alice" gets deployment info for the deployment
    Then "Alice" should receive deployment info with status and manifest from store
    And allocations should be empty or contain minimal info
