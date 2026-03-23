Feature: Transforms
  Define transforms in packages.

  @1.13.0
  Scenario: Installer installs the transform included in package
    Given the "transform" package is installed
     Then there is a transform "logs-transform.metadata_united-*"

  @2.10.0
  Scenario: Configure aliases for transforms
    Given the "transform_aliases" package is installed
     Then there is a transform "logs-transform.metadata_united-*"
      And there is a transform alias "transform_aliases_transform_alias"
