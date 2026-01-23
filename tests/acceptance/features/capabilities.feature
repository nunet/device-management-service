Feature: Capabilities Management
  Grant, delegate, revoke, and anchor behavior-level capabilities across users.

  Background:
    Given the following nodes
      | nodes | role | onboarded | org   |
      | Alice | SP   | false     | nunet |
      | Bob   | SP   | false      | nunet |

  Scenario Outline: Revoke a previously granted capability
    Given "Alice" says hello to "Bob"
    Then "Bob" should respond with his <DID>
    When "Bob" revokes permission from "Alice" via "nunet"
    And "Alice" says hello to "Bob"
    Then "Bob" should not respond with his <DID>
