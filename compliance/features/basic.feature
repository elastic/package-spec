Feature: Minimal packages
  Basic tests with minimal packages

  @1.0.0
  Scenario: Integration package can be installed
   Given an "integration" package
     And the package is installed

  @2.6.0
  Scenario: Input package can be installed
   Given an "input" package
     And the package is installed
