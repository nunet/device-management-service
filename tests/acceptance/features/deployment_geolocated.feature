Feature: Deployment Geolocated
  As a Service Provider
  I want deployments with location constraints to run only on eligible nodes
  So that deployments follow geographic policies

  Background:
    Given the following nodes:
      | nodes | role | onboarded | org   | continent | country |
      | Alice | SP   | false     | nunet |           |         |
      | Bob   | CP   | true      | nunet | NA        | US      |
      | Carol | CP   | true      | nunet | EU        | DE      |
      | Dave  | CP   | true      | nunet | AS        | CH      |

  @wip
  @complexity:high
  Scenario Outline: Deployment is executed only on a node satisfying location constraints
    When the <ensemble> is submitted
    Then the deployment status should be <status>
    And the following nodes in the ensemble should be allocated on the expected running nodes:
      | ensemble_node   | running_node   |
      | node1           | <running_node> |
    And the deployment should produce the following outputs:
      | ensemble_node | allocation | log                |
      | node1         | alloc1     | Hello from Docker! |

    Examples:
      | ensemble                        | description                                                 | status    | running_node  |
      | docker_hello_constraint_ch.yaml | a node with location defined to continent AS and country CH | Completed | Dave          |
      | docker_hello_constraint_de.yaml | a node with location defined to continent EU and country DE | Completed | Carol         |

  @wip
  @complexity:high
  Scenario Outline: Deployment is distributed across two eligible nodes based on location constraints
    When the <ensemble> is submitted
    Then the deployment status should be <status>
    And the following nodes in the ensemble should be allocated on the expected running nodes:
      | ensemble_node | running_node   |
      | node1         | <running_node1> |
      | node2         | <running_node2> |
    And the deployment should produce the following outputs:
      | ensemble_node | allocation | log                |
      | node1         | alloc1     | Hello from Docker! |
      | node2         | alloc2     | Hello from Docker! |

    Examples:
      | ensemble                           | description                                        | status    | running_node1 | running_node2 |
      | docker_hello_constraint_ch_de.yaml | two nodes with location defined to AS/CH and EU/DE | Completed | Dave          | Carol         |

  @wip
  @complexity:high
  Scenario Outline: Deployment fails when one or more nodes have no matching running nodes for geographic constraints
    When the <ensemble> is submitted
    Then the deployment should fail with error "<error>"

    Examples:
      | ensemble                           | description                                                 | error                      |
      | docker_hello_constraint_za.yaml    | a node with location defined to continent AF and country ZA | failed to get orchestrator |
      | docker_hello_constraint_za_us.yaml | two nodes with location defined to AF/ZA and NA/US          | failed to get orchestrator |

