Feature: Synthetic source
  Support synthetic source

  # Requires a new elastic-package release to be able to use -C flag  to run tests
  # of a package located in another directory (https://github.com/elastic/elastic-package/pull/1914)
  @skip
  @3.2.0
  Scenario: Installer leverages source true
   Given the "logs_synthetic_mode" package is installed
     And a policy is created with "logs_synthetic_mode" package
    Then index template "logs-logs_synthetic_mode.synthetic" has a field "decision_id" with "store:true"
