Feature: Allocation Running on Subnet
  As a Service Provider
  I want to launch services on peers
  And the services can communicate with each other

  Background:
    Given the following nodes
      | nodes   | role | onboarded | org   |
      | Alice   | SP   | false     | nunet |
      | Bob     | CP   | true      | nunet |
      | Charlie | CP   | true      | nunet |

  Scenario: Allocations communicating on the same subnet
    Given "Alice" has services deployed on "Bob" and "Charlie"
    When "Bob" service tries to communicate with "Charlie"
    Then they should get a OK response
