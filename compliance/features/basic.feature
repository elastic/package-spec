Feature: Minimal packages
  Basic tests with minimal packages

  @1.0.0
  Scenario: Integration package can be installed
   Given the "basic_integration" package is installed
    Then there is an index template for pattern "metrics-basic_integration.foo-*"

  @2.6.0
  Scenario: Input package can be installed
   Given the "basic_input" package is installed
    Then there is an index template for pattern "metrics-basic_input.foo-*"
