Feature: Allocation Running on Subnet

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
