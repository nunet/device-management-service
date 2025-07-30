@wip
Feature: Resource Management
  As a Computer Provider
  I want to manage the computational resources of my machine (onboard/offboard)
  And ensure deployments running on my machine stay within declared limits
  And ensure resources are correctly freed once the deployment finishes

  Background:
    Given the following nodes:
      | nodes | role | onboarded | org   |
      | Alice | SP   | false     | nunet |
      | Bob   | CP   | false     | nunet |

  @wip
  @complexity:low
  Scenario: Onboard resources to the network
    When "Bob" onboards with:
      | CPU | RAM | Disk |
      | 2   | 4GB | 10GB |
    Then the onboard status should be onboarded
    And the resources onboarded should reflect the correct advertised resources

  @wip
  @complexity:low
  Scenario: Offboard resources from the network
    Given "Bob" is onboarded with:
      | CPU | RAM | Disk |
      | 2   | 4GB | 10GB |
    When "Bob" offboards
    Then the onboard status should be offboarded
    And the resources onboarded should no longer advertise any resources

  @wip
  @complexity:low
  Scenario: Deployment is reject due to insufficient resources
    Given "Bob" is onboarded with:
      | CPU | RAM | Disk |
      | 2   | 4GB | 10GB |
    When "Alice" deployment requiring:
      | CPU | RAM | Disk |
      | 4   | 2GB | 5BG |
      | 1   | 6GB | 5GB |
      | 1   | 2GB | 20GB |
    is submitted to a network with only "Bob"
    Then the deployment should be rejected due to insufficient resources

  @wip
  @complexity:low
  Scenario: Targeting deployment is reject due to insufficient resources
    Given "Bob" is onboarded with:
      | CPU | RAM | Disk |
      | 2   | 4GB | 10GB |
    When "Alice" deployment requiring:
      | CPU | RAM | Disk |
      | 4   | 2GB | 5BG  |
      | 1   | 6GB | 5GB  |
      | 1   | 2GB | 20GB |
    is submitted to the network targeting "Bob"
    Then the deployment should be rejected due to insufficient resources

  @wip
  @complexity:low
  Scenario: Deployment allocates the specified resources
    Given "Bob" is onboarded with:
      | CPU | RAM | Disk |
      | 2   | 4GB | 10GB |
    And "Alice" deployment requiring:
      | CPU | RAM | Disk |
      | 2   | 2GB | 5GB  |
    is deployed on "Bob"
    Then the allocated resource status on "Bob" should reflect the updated availability
    And the resources should be allocated on "Bob"

  @wip
  @complexity:low
  Scenario: Deployment releases the allocated resources
    Given "Bob" is onboarded with:
      | CPU | RAM | Disk |
      | 2   | 4GB | 10GB |
    And "Alice" deployment requiring:
      | CPU | RAM | Disk |
      | 2   | 2GB | 5GB  |
    is deployed on "Bob"
    When "Alice" deployment is completed
    Then the free resource status on "Bob" should reflect the full availability
    And the resources should be released on "Bob"

  @wip
  @complexity:medium
  Scenario: Deployment does not use more CPU than specified in the ensemble
    Given "Bob" is onboarded with:
      | CPU | RAM | Disk |
      | 2   | 4GB | 10GB |
    When "Alice" deployment requiring:
      | CPU | RAM | Disk |
      | 2   | 2GB | 5GB  |
    is deployed on "Bob"
    And "Alice" deployment attempts to use more CPU than specified in the ensemble
    Then "Bob" should enforce the CPU limit as defined in the ensemble

  @wip
  @complexity:medium
  Scenario: Deployment does not use more RAM than specified in the ensemble
    Given "Bob" is onboarded with:
      | CPU | RAM | Disk |
      | 2   | 4GB | 10GB |
    When "Alice" deployment requiring:
      | CPU | RAM | Disk |
      | 2   | 2GB | 5GB  |
    is deployed on "Bob"
    And "Alice" deployment attempts to use more RAM than specified in the ensemble
    Then "Bob" should enforce the RAM limit as defined in the ensemble

  @wip
  @complexity:medium
  Scenario: Deployment does not use more disk than specified in the ensemble
    Given "Bob" is onboarded with:
      | CPU | RAM | Disk |
      | 2   | 4GB | 10GB |
    When "Alice" deployment requiring:
      | CPU | RAM | Disk |
      | 2   | 2GB | 5GB  |
    is deployed on "Bob"
    And "Alice" deployment attempts to use more disk than specified in the ensemble
    Then "Bob" should enforce the disk limit as defined in the ensemble

  @wip
  @complexity:low
  Scenario: Retrieve the hardware specifications

  @wip
  @complexity:low
  Scenario: Retrieve the current hardware usage
