Feature: Runtime fields
  Configure fields in a package as runtime fields.

  @2.8.0
  Scenario: Installer leverages runtime parameter
    When "package-with-runtime-fields" is installed.
    Then index template for "package-with-runtime-fields" includes runtime fields.
