Feature: P2P connectivity through Network Address Translation (NAT)
  As a distributed system
  I want to test how nodes behind different NATs communicate
  So that I can verify NAT traversal and relay behavior

  Scenario: Two nodes behind different NATs connect via relay
    # Step 1: Setup Alice and Bob behind NAT
    Given I have 2 DMS nodes on isolated NAT networks
      | name  | network    |
      | Alice | nat-net-1  |
      | Bob   | nat-net-2  |
    # Step 2: Try direct connection - should fail due to NAT
    And "Alice" attempts to connect to "Bob"
    Then the connection should fail if no relay
    # Step 3: Wait for AutoNAT/AutoRelay to detect NAT and establish relay circuits
    And Target "Bob" waits to perform AutoNAT and obtain relay circuits
    # Step 4: Verify relay addresses are now advertised
    And "Alice" should have a relay address advertised
    And "Bob" should have a relay address advertised
    # Step 5: Connect via auto-discovered relay
    And "Alice" attempts to connect to "Bob"
    Then the connection should succeed via relay


