@wip
Feature: Deployment Launch
  Enables launching single or multiple deployments across peers.

  Scenario: Launch a deployment on any available peer in the network
    Given I am connected to the network
    And there is at least one available peer
    When I launch a deployment without selecting a specific peer
    Then the deployment should be assigned to an available peer
    And it should start successfully

  Scenario: Launch a deployment on a selected peer
    Given I am connected to the network
    And a specific peer is connected to the network and available
    When I launch a deployment targeting that peer
    Then the deployment should be assigned to the selected peer
    And it should start successfully

  Scenario: Launch multiple deployments on the same peer
    Given a peer is available and has enough resources to support concurrent deployments
    When I launch multiple deployments targeting that peer
    Then all deployments should be assigned to that peer
    And they should start successfully

  Scenario: Launch multiple deployments on different peers
    Given multiple peers are available and ready
    When I launch deployments targeting different peers
    Then each deployment should be assigned to its respective peer
    And all deployments should start successfully

  Scenario: Launch multiple deployments communicating on the same peer
    Given multiple deployments are running concurrently on the same peer
    When inter-deployment communication is initiated
    Then the communication should be established successful

  Scenario: Launch multiple deployments communicating on different peers
    Given multiple deployments are running on different peers
    When inter-deployment communication is initiated
    Then the communication should be established successfully across peers
