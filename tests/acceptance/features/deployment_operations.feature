Feature: Deployment Operations
  View, manage, and interact with active or failed deployments.

  Background:
    Given the following nodes
      | nodes   | role | onboarded | org   |
      | Alice   | SP   | false     | nunet |
      | Bob     | CP   | true      | nunet |

  @wip
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

  @wip
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

  @wip
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

  @complexity:medium
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
