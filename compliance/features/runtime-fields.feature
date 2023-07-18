Feature: Runtime fields
  Configure fields in a package as runtime fields.

  @2.8.0
  Scenario: Installer leverages runtime parameter
   Given an "integration" package
    When the package has "runtime-fields"
     And the package is installed
    Then index template "includes runtime fields"
