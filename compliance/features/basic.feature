Feature: Basic package types support
  Basic tests with minimal packages

  @1.0.0
  Scenario: Integration package can be installed
   Given the "basic_integration" package is installed
     And a policy is created with "basic_integration" package
    Then there is an index template "metrics-basic_integration.foo" with pattern "metrics-basic_integration.foo-*"

  @2.6.0
  Scenario: Input package can be installed
   Given the "basic_input" package is installed
     And a policy is created with "basic_input" package, "test" template, "test" input, "logfile" input type and dataset "spec.input_test"
    Then there is an index template "logs-spec.input_test" with pattern "logs-spec.input_test-*"

  @3.3.0
  Scenario: Basic content package can be installed
   Given the "basic_content" package is installed
     And prebuilt detection rules are loaded
    Then there is a dashboard "basic_content-dashboard-abc-1"
     And there is a detection rule "12cea9e9-5766-474d-a9dc-34ef7c7677c7"

  @3.5.0
  Scenario: Content package can be installed
   Given the "good_content" package is installed
     And prebuilt detection rules are loaded
    Then there is a dashboard "good_content-dashboard-abc-1"
     # Missing support in Kibana (Fleet) https://github.com/elastic/kibana/pull/186974
     And there is an SLO "good_content-slo-abc-1"
     And there is a detection rule "12cea9e9-5766-474d-a9dc-34ef7c7677c6"
