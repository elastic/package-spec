Feature: ES|QL
  Features related to ES|QL

  @3.4.2
  Scenario: Installer leverages lookup index mode
   Given the "good_lookup_index" package is installed
     And a policy is created with "good_lookup_index" package and "0.1.4" version
    Then index template "logs-good_lookup_index.foo-template" is configured for "lookup index mode"

  @3.6.7
  @skip
  # Pending Fleet/elastic-package support for ES|QL views installation
  Scenario: Content package with ES|QL view installs the view
   Given the "good_content" package is installed
    Then there is an ES|QL view "good_content-my_view"
