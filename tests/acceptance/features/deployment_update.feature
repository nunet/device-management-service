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

  Scenario: Add a node to running ensemble
    Given "Alice" has service deployed on "Bob"
    And "Alice" deployment is running
    When "Alice" adds "Charlie" to the deployment
    Then "Alice" deployment should be running on "Charlie"
    And "Alice" deployment should be running on "Bob"

  Scenario: Remove a node from running ensemble
    Given "Alice" has services deployed on "Bob" and "Charlie"
    And "Alice" deployment is running
    When "Alice" removes "Charlie" from the deployment
    Then "Alice" deployment should not be running on "Charlie"
    But "Alice" deployment should be running on "Bob"

  Scenario: Update node in running ensemble
    Given "Alice" has service deployed on "Bob"
    And "Alice" deployment is running
    When "Alice" updates deployment to run on "Charlie"
    Then "Alice" deployment should be running on "Charlie"
    But "Alice" deployment should not be running on "Bob"

  Scenario: Add allocation to running ensemble
    Given "Alice" has 1 allocation deployed on "Bob"
    And "Alice" deployment is running
    When "Alice" adds 1 allocation to "Bob"
    Then "Alice" deployment should have 2 allocations running on "Bob"

  Scenario: Remove allocation from running ensemble
    Given "Alice" has 2 allocations deployed on "Bob"
    And "Alice" deployment is running
    When "Alice" removes 1 allocation from "Bob"
    Then "Alice" deployment should have 1 allocation running on "Bob"
