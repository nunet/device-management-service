@wip
Feature: Deployment Cancellation
  Cancel deployments in various lifecycle stages and error conditions.

  Scenario: Cancel an ongoing deployment
    Given a deployment is currently running on a peer
    When I request cancellation of the deployment
    Then the deployment should begin termination promptly
    And all associated resources should be released
    And the deployment status should be marked as cancelled

  Scenario: Attempt to cancel a deployment on an unavailable peer
    Given a deployment is running on a peer
    And the peer becomes unreachable
    When I request cancellation of the deployment
    Then the system should retry cancellation for a limited time
    And I should be notified that the cancellation is pending or failed

  # TODO: Check if this is a valid scenario
  Scenario: Attempt to cancel a deployment that has already completed
    Given a deployment has already finished execution
    When I request cancellation of the deployment
    Then I should receive a message indicating the deployment is already complete
