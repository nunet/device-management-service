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
  Scenario Outline: Deploy a service and send a request
    When the <ensemble> is submitted
    Then the deployment status should be <status>
    And the deployment should produce the following outputs:
      | ensemble_node | allocation | log           |
      | node1         | alloc1     | nginx started |
    When a request is sent to the service on <running_node>
    Then the request response status should be <response_status>

    Examples:
      | ensemble          | description         | status  | running_node | response_status |
      | docker_nginx.yaml | one node runs nginx | Running | Bob          | 200             |

