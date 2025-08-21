Feature: Deployment
  As a Service Provider
  I want to deploy my computation on other nodes
  So that I don't have to use my machine

  Background:
    Given the following nodes
      | nodes   | role | onboarded | org   |
      | Alice   | SP   | false     | nunet |
      | Bob     | CP   | true      | nunet |
      | Charlie | CP   | true      | nunet |

  Scenario Outline: Retrieve output from execution
    Given "Alice" has deployed <ensemble> on "Bob"
    When "Alice" deployment is completed
    Then "Alice" ensemble should return <output>

  Examples:
    | ensemble            | output               |
    | "docker_hello.yaml" | "Hello from Docker!" |

  Scenario: Remove node in running ensemble
    Given "Alice" has services deployed on "Bob" and "Charlie"
    And "Alice" deployment is running
    When "Alice" updates deployment to remove "Charlie"
    Then "Alice" deployment should be running on "Bob"
    But "Alice" deployment should not be running on "Charlie"

  Scenario: Add node in running ensemble
    Given "Alice" has service deployed on "Bob"
    And "Alice" deployment is running
    When "Alice" updates deployment to add "Charlie"
    Then "Alice" deployment should be running on "Charlie"
    And "Alice" deployment should be running on "Bob"

  Scenario: Add allocation to running ensemble
    Given "Alice" has deployed ensemble with 1 allocation on "Bob"
    And "Alice" deployment is running
    When "Alice" updates deployment to add 1 allocation
    Then "Alice" deployment should have 2 allocations running on "Bob"

  Scenario: Remove allocation from running ensemble
    Given "Alice" has deployed ensemble with 2 allocations on "Bob"
    And "Alice" deployment is running
    When "Alice" updates deployment to remove 1 allocation
    Then "Alice" deployment should have 1 allocation running on "Bob"
