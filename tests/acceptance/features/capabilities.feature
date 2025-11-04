Feature: Capabilities Management
  Grant, delegate, revoke, and anchor behavior-level capabilities across users.

  Background:
    Given the following nodes
      | nodes | role | onboarded | org   |
      | Alice | SP   | false     | nunet |
      | Bob   | CP   | true      | nunet |

  Scenario Outline: Revoke a previously granted capability
    Given "Alice" has deployed <ensemble> on "Bob"
    When "Alice" deployment is completed
    Then "Alice" ensemble should return <output>
    When "Bob" revokes a token from "Alice" via "nunet"
    And "Alice" has deployed <ensemble> on "Bob"
    Then "Alice" deployment should not succeed on "Bob"

  Examples:
    | ensemble            | output               |
    | "docker_hello.yaml" | "Hello from Docker!" |

