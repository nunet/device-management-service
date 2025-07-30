@wip
Feature: Deployment Update
  As a Service Provider  
  I want to dynamically adjust the nodes and allocations of an ensemble  
  So that I can scale my deployment and control its execution with flexibility  

  Background:  
    Given the following nodes
      | nodes   | role | onboarded | org   |
      | Alice   | SP   | false     | nunet |
      | Bob     | CP   | true      | nunet |
      | Charlie | CP   | true      | nunet |

  @wip
  @complexity:medium
  Scenario: Add a node to the ensemble  
    Given "Alice" has services deployed on "Bob"
    When "Alice" adds one more node to the ensemble
    And "Alice" submits the updated ensemble
    Then the services should be deployed on "Charlie"

  @wip
  @complexity:medium
  Scenario: Remove a node from the ensemble  
    Given "Alice" has services deployed on "Bob" and "Charlie"
    When "Alice" removes "Charlie" from the ensemble
    And "Alice" submits the updated ensemble
    Then no service should remain deployed on "Charlie"

  @wip
  @complexity:medium
  Scenario: Add an allocation to a node  
    Given "Alice" has an active deployment on "Bob"
    When "Alice" adds a new allocation to "Bob"
    And "Alice" submits the updated ensemble
    Then a new allocation should be deployed on "Bob"

  @wip
  @complexity:medium
  Scenario: Remove an allocation from a node  
    Given "Alice" has two allocations deployed on "Bob"
    When "Alice" removes one allocation from "Bob"
    And "Alice" submits the updated ensemble
    Then the specific allocation removed should be terminated on "Bob"

