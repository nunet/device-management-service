@wip
Feature: Deployment in an IP Network
  As a Service Provider
  I want to deploy services in two nodes or more
  And I want the allocations running in each node to be able to communicate with each other

  @wip
  @complexity:medium
  Scenario Outline: Check communication between allocations
    Given "Alice" has deployed <ensemble> on "Bob" and "Charlie"
    When "Alice" deployment is completed
    Then "Alice" ensemble should have 2 allocations with IP addresses
    And "Bob" and "Charlie" allocations should be able to communicate with each other

  @wip
  @complexity:medium
  Scenario Outline: Check communication between allocations after redeployment
    Given "Alice" has deployed <ensemble> on "Bob" and "Charlie"
    When "Alice" deployment is completed
    Then "Alice" stops the deployment
    Then "Alice" starts the deployment again
    Then "Alice" ensemble should have 2 allocations with IP addresses
    And "Bob" and "Charlie" allocations should be able to communicate with each other

  @wip
  @complexity:medium
  Scenario Outline: Check communication over DNS between allocations
    Given "Alice" has deployed <ensemble> on "Bob" and "Charlie"
    When "Alice" deployment is completed
    Then "Alice" ensemble should have 2 allocations with IP addresses and DNS addresses
    And "Bob" and "Charlie" allocations should be able to resolve each other's DNS addresses

  @wip
  @complexity:medium
  Scenario Outline: Check communication over DNS between allocations after redeployment
    Given "Alice" has deployed <ensemble> on "Bob" and "Charlie"
    When "Alice" deployment is completed
    Then "Alice" stops the deployment
    Then "Alice" starts the deployment again
    Then "Alice" ensemble should have 2 allocations with IP addresses and DNS addresses
    And "Bob" and "Charlie" allocations should be able to resolve each other's DNS addresses

  @wip
  @complexity:medium
  Scenario: Connect to a running deployment via SSH
