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
    When "Alice" retrieves its libp2p address
    And "Bob" retrieves its libp2p address
    # Step 2: Try direct connection - should fail due to NAT
    And "Alice" attempts to connect to "Bob"
    Then the connection should fail due to NAT
    # Step 3: Spin up relay node
    When I create a relay node "Relay" on public network
    And "Relay" retrieves its libp2p address
    # Step 4: Connect to relay
    And "Alice" connects to relay "Relay"
    And "Bob" connects to relay "Relay"
    # Step 5: Wait for AutoNAT/AutoRelay to detect NAT and establish relay circuits
    And I wait 90 seconds for AutoNAT and relay circuits to establish
    # Step 6: Verify relay addresses are now advertised
    And "Alice" should have a relay address advertised
    And "Bob" should have a relay address advertised
    # Step 7: Connect via auto-discovered relay
    And "Alice" attempts to connect to "Bob"
    Then the connection should succeed via relay


