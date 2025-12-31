Feature: Deployment
  As a Service Provider
  I want to deploy my computation on other nodes
  So that I don't have to use my machine

  Background:
    Given the following nodes
      | nodes   | role | onboarded | org   |
      | Alice   | SP   | false     | nunet |
      | Bob     | CP   | true      | nunet |

  Scenario Outline: Retrieve output from execution
    Given "Alice" has deployed <ensemble> on "Bob"
    When "Alice" deployment is completed
    Then "Alice" ensemble should return <output>

  Examples:
    | ensemble            | output               |
    | "docker_hello.yaml" | "Hello from Docker!" |
