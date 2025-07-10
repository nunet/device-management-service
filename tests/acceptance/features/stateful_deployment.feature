@wip
Feature: Stateful Deployment
  Handle deployments with data persistence across reboots or environments.

  Scenario: Persist deployment data to a local volume (disk)
    Given a deployment is stateful and running
    And it generates internal data during execution
    When it completes
    Then it should persist its data to durable storage

  Scenario: Launch a new deployment restoring from a local volume
    Given a previous saved data exists for the deployment
    When a new deployment starts
    Then it should retrieve and apply the saved state
    And continue execution using the restored data

  Scenario: Attempt to restore data from an invalid or missing storage path
    Given there is no valid saved data available
    When I attempt to launch a deployment using a saved state
    Then the deployment should fail to start
    And I should receive an error indicating the data is unavailable or corrupted

