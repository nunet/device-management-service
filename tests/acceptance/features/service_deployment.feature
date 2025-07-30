Feature: Service Deployment
  As a Service Provider
  I want to launch a service on a peer
  So that I can send a request to it

  Background:
    Given the following nodes:
      | nodes | role | onboarded | org   |
      | Alice | SP   | false     | nunet |
      | Bob   | CP   | true      | nunet |

  @wip
  @complexity:medium
  Scenario Outline: Launch a service in a node and send a request
    When the <ensemble> is submitted
    Then the deployment status should be <status>
    And the deployment should produce the following outputs:
      | ensemble_node | allocation | log           |
      | node1         | alloc1     | nginx started |
    And the list of all deployments on the node should has <deploy_amount> deployments
    And the list of all allocations on the node should has <alloc_amount> allocations
    When a request is sent to the service on <running_node>
    Then the request response status should be <response_status>

    Examples:
      | ensemble          | description         | status  | running_node | response_status | deploy_amount | alloc_amount |
      | docker_nginx.yaml | one node runs nginx | Running | Bob          | 200             | 1             | 1            |

  @wip
  @complexity:medium
  Scenario Outline: Launch multiple services in the same deployment and send a request
    When the <ensemble> is submitted
    Then the deployment status should be <status>
    And the deployment should produce the following outputs:
      | ensemble_node | allocation | log           |
      | node1         | alloc1     | nginx started |
      | node1         | alloc2     | nginx started |
    And the list of all deployments on the node should has <deploy_amount> deployments
    And the list of all allocations on the node should has <alloc_amount> allocations
    When a request is sent to each service on <running_node>
    Then the request response status should be <response_status>

    Examples:
      | ensemble          | description                                | status  | running_node | response_status | deploy_amount | alloc_amount |
      | docker_nginx.yaml | one node runs two nginx in different ports | Running | Bob          | 200             | 1             | 2            |

  @wip
  @complexity:medium
  Scenario Outline: Launch multiple services in different deployments on the same peer
    When the <ensemble> is submitted twice
    Then the deployments status should be <status>
    And the deployments should produce the following outputs:
      | ensemble_node | allocation | log           |
      | node1         | alloc1     | nginx started |
    And the list of all deployments on the node should has <deploy_amount> deployments
    And the list of all allocations on the node should has <alloc_amount> allocations
    When a request is sent to each service on <running_node>
    Then the request response status should be <response_status>

    Examples:
      | ensemble          | description         | status  | running_node | response_status | deploy_amount | alloc_amount |
      | docker_nginx.yaml | one node runs nginx | Running | Bob          | 200             | 2             | 2            |

