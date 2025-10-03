Feature: Contract Management Simple
  As a Service Provider
  I want to create contracts with Compute Providers
  So that I can guarantee exclusive access to specific resources and ensure secure, isolated deployments

  # SP -> Service Provider
  # CP -> Compute Provider
  # CH -> Contract Host

  Background:
    Given the following nodes
      | nodes   | role | onboarded | org   |
      | Alice   | SP   | false     | nunet |
      | Bob     | CP   | true      | nunet |
      | Charlie | CH   | false     | nunet |

  Scenario: Create a contract
    Given "Alice" requests a contract with "Bob" through "Charlie"
    When "Bob" accepts the contract
    Then a contract is created between "Alice" and "Bob"

  Scenario: Make deployment with contract
    Given "Alice" requests a contract with "Bob" through "Charlie"
    And "Bob" accepts the contract
    When "Alice" deploys a task on "Bob" with their contract
    Then "Alice" contract should be active

  Scenario: Make deployment without signed contract
    Given "Alice" requests a contract with "Bob" through "Charlie"
    And "Bob" does not accept the contract
    When "Alice" deploys a task on "Bob" with their contract
    Then "Alice" deployment should fail

  Scenario: Mark contract as completed
    Given "Alice" has an active contract with "Bob" created through "Charlie"
    When "Alice" marks contract as completed
    Then "Bob" should see contract as completed

  Scenario: Terminate a contract
    Given "Alice" has an active contract with "Bob" created through "Charlie"
    When "Alice" requests to terminate the contract
    Then "Bob" should see contract as terminated
