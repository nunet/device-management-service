@wip
Feature: Observability system integration and behavior
  The system must provide structured logging, event emission, tracing,
  and dynamic configuration capabilities through the observability package.

  Background:
    Given observability is initialized with:
      | host        | did            | apm_enabled |
      | LibP2PHost  | did:nunet:test | true        |

  Scenario: Produce a structured console log
    When the system logs an info message "System started"
    Then the console output should include the message with level "INFO"

  Scenario: Log message with labels for routing
    When the system logs a message "Allocation created" with labels:
      | label      |
      | allocation |
    Then the message should be routed according to the label configuration

  Scenario: Emit a custom event to the internal event bus
    When the system emits an event with type "DeploymentCompleted" and payload:
      | key        | value   |
      | serviceID  | svc-001 |
      | status     | success |
    Then the event bus should receive an event of type "DeploymentCompleted"

  Scenario: Start a distributed trace with a transaction name
    When a trace is started with name "process_order" and parameter:
      | key     | value  |
      | orderID | 123456 |
    Then a new APM transaction should be created with name "process_order"

  Scenario: Update the log level dynamically
    Given the current log level is "INFO"
    When the log level is updated to "DEBUG"
    Then debug logs should be emitted

  Scenario: Enable no-op mode to disable all observability
    When no-op mode is enabled
    Then no logs, traces or events should be emitted

  Scenario: Set a new Elasticsearch endpoint at runtime
    When the Elasticsearch endpoint is updated to "https://es.example.com:9200"
    Then logs should be sent to "https://es.example.com:9200"

  Scenario: Shutdown observability gracefully
    When observability is shut down
    Then all logs and traces should be flushed
    And event emitters should be closed

