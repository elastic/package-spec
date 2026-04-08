Feature: Transforms
  Define transforms in packages.

  @1.13.0
  Scenario: Installer installs the transform included in package
    Given the "transform" package is installed
     Then there is a transform "logs-transform.metadata_united-*"

  @2.10.0
  Scenario: Configure aliases for transforms
    Given the "transform_aliases" package is installed
     Then there is a transform "logs-transform_aliases.metadata_united-*"
      And the transform "logs-transform_aliases.metadata_united-*" has alias "transform_aliases_transform_alias" configured
