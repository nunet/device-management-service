Feature: Allocation Running on Subnet

  Scenario: Allocations communicating on the same subnet
    Given "Alice" has services deployed on "Bob" and "Charlie"
    When "Bob" service tries to communicate with "Charlie"
    Then they should get a OK response
