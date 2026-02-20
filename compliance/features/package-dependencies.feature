Feature: Package dependencies support
  Integration packages can declare dependencies on input and content packages

  @3.6.0
  Scenario: Integration package with dependencies installs required packages
   Given the "good_requires" package is installed
    Then the content packages "good_requires" require are installed
