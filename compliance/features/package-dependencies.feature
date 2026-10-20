Feature: Package dependencies support
  Integration packages can declare dependencies on input and content packages

  # The support for this feature was added in v9.4.0 https://github.com/elastic/kibana/pull/256700 behind a feature flag
  # In v9.5.0 this feature flag was enabled https://github.com/elastic/kibana/pull/263762
  # As spec version 3.6 runs in v9.4 and v9.5, this test could not run in both, setting for now spec 3.7.0
  # Tested locally that it worked with v9.5.6-SNAPSHOT, so the scenario is moved to spec 3.7.0 for that reason
  @3.7.0
  Scenario: Integration package with dependencies installs required packages
   Given the "apache_otel" package is uninstalled
     And the "good_requires" package is installed
    Then the content packages "good_requires" require are installed
