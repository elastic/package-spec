Feature: Runtime fields
  Configure fields in a package as runtime fields.

  @2.8.0
  Scenario: Installer leverages runtime parameter
   Given the "runtime_fields" package is installed
    Then index template "metrics-runtime_fields.foo" includes "runtime fields"
