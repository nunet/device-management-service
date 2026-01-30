Feature: Capabilities Revocation Broadcast
  Broadcast revocation tokens to all nodes in the network

  Background:
    Given the following nodes
      | nodes | role | onboarded | org   |
      | Alice | SP   | false     | nunet |
      | Bob   | CP   | false      | nunet |

  Scenario Outline: Revoke a previously granted capability and broadcast revocation
    Given "Alice" says hello to "Bob"
    Then "Bob" should respond with his <DID>
    When "Nunet" revokes a token for "Alice" and "Alice" broadcasts it
    And "Alice" says hello to "Bob"
    Then "Bob" should not respond with his <DID>