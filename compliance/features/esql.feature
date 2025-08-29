Feature: ES|QL
  Features related to ES|QL

  @3.4.2
  Scenario: Installer leverages lookup index mode
   Given the "good_lookup_index" package is installed
     And a policy is created with "good_lookup_index" package and "0.1.4" version
    Then index template "logs-good_lookup_index.foo-template" is configured for "lookup index mode"
