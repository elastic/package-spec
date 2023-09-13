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
     And a policy is created with "basic_input" package, "test" template, "test" input, "logfile" input type and dataset "spec.input-test"
    Then there is an index template "logs-spec.input-test" with pattern "logs-spec.input-test-*"
