Feature: Runtime fields
  Configure fields in a package as runtime fields.

  @2.8.0
  Scenario: Installer leverages runtime parameter
   Given the "runtime_fields" package is installed
     And a policy is created with "runtime_fields" package
    Then index template "metrics-runtime_fields.foo" has a field "foo.runtime_field" with "runtime:true"
