Feature: Task Deployment
  As a Service Provider
  I want to deploy tasks across one or multiple peers
  So that I don't have to use my machine

  Background:
    Given the following nodes:
      | nodes | role | onboarded | org   |
      | Alice | SP   | false     | nunet |
      | Bob   | CP   | true      | nunet |
      | Carol | CP   | true      | nunet |

  @wip
  Scenario Outline: Launch a deployment on any available peer in the network
    When the <ensemble> is submitted
    Then the deployment status should be <status>
    And the deployment should produce the following outputs:
      | node   | allocation | log                |
      | node1  | alloc1     | Hello from Docker! |

    Examples:
      | ensemble          | description                         | status    |
      | docker_hello.yaml | one node without targeting any peer | Completed |

  @wip
  Scenario Outline: Launch a deployment on a target peer
    When the <ensemble> is submitted
    Then the deployment status should be <status>
    And the following nodes in the ensemble should be allocated on the expected running nodes:
      | ensemble_node | running_node   |
      | node1         | <running_node> |
    And the deployment should produce the following outputs:
      | ensemble_node | allocation | log                |
      | node1         | alloc1     | Hello from Docker! |

    Examples:
      | ensemble          | description                   | status    | running_node |
      | docker_hello.yaml | one node targeting Bob's peer | Completed | Bob          |

  @wip
  Scenario Outline: Launch multiple deployments on the same peer
    When the <ensemble> is submitted
    Then the deployment status should be <status>
    And the deployment should produce the following outputs:
      | ensemble_node | allocation | log                |
      | node1         | alloc1     | Hello from Docker! |
      | node1         | alloc2     | Hello from Docker! |

    Examples:
      | ensemble          | description                              | status    |
      | docker_hello.yaml | two allocations running on the same node | Completed |

  @wip
  Scenario Outline: Launch multiple deployments on different peers
    When the <ensemble> is submitted
    Then the deployment status should be <status>
    And the deployment should produce the following outputs:
      | node   | allocation | log                |
      | node1  | alloc1     | Hello from Docker! |
      | node2  | alloc2     | Hello from Docker! |

    Examples:
      | ensemble          | description                                | status    |
      | docker_hello.yaml | two allocations running on different nodes | Completed |

