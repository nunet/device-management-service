@wip
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
