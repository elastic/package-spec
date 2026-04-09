Feature: Data stream dataset configuration
  Verify data_stream.dataset is correctly set in compiled policies and index templates

  @1.0.0
  Scenario: Integration package uses default dataset
    Given the "basic_integration" package is installed
      And a policy is created with "basic_integration" package
     Then there is an index template "metrics-basic_integration.foo" with pattern "metrics-basic_integration.foo-*"
     # Compiled policy doesn't include data_stream.dataset for integration packages without a data_stream.dataset variable
     # Then the compiled policy has dataset "basic_integration.foo" for "sample/metrics" input type

  @1.0.0
  Scenario: Integration package with dataset variable uses default dataset
    Given the "dataset_integration" package is installed
      And a policy is created with "dataset_integration" package
     Then the compiled policy has dataset "dataset_integration.generic" for "logfile" input type
     # Index template not checked for integration packages: https://github.com/elastic/kibana/issues/160775
     # Then there is an index template "logs-dataset_integration.generic" with pattern "logs-dataset_integration.generic-*"

  @2.6.0
  Scenario: Input package uses default dataset
    Given the "basic_input" package is installed
      And a policy is created with "basic_input" package, "1.0.0" version, "test" template, "test" input, "logfile" input type and dataset ""
     Then the compiled policy has dataset "test" for "logfile" input type
      And there is an index template "logs-test" with pattern "logs-test-*"

  @2.6.0
  Scenario: Input package dataset can be overridden
    Given the "basic_input" package is installed
      And a policy is created with "basic_input" package, "1.0.0" version, "test" template, "test" input, "logfile" input type and dataset "custom.input_test"
     Then the compiled policy has dataset "custom.input_test" for "logfile" input type
      And there is an index template "logs-custom.input_test" with pattern "logs-custom.input_test-*"

  @3.5.0
  Scenario: Input OTel package uses default dataset
    Given the "good_input_otel" package is installed
      And a policy is created with "good_input_otel" package, "0.0.1" version, "httpcheckreceiver" template, "httpcheckreceiver" input, "otelcol" input type and dataset ""
     Then the compiled policy has dataset "httpcheckreceiver" for "otelcol" input type
      And there is an index template "metrics-httpcheckreceiver" with pattern "metrics-httpcheckreceiver.otel-*"

  @3.5.0
  Scenario: Input OTel package dataset can be overridden
    Given the "good_input_otel" package is installed
      And a policy is created with "good_input_otel" package, "0.0.1" version, "httpcheckreceiver" template, "httpcheckreceiver" input, "otelcol" input type and dataset "custom.otel_test"
     Then the compiled policy has dataset "custom.otel_test" for "otelcol" input type
      And there is an index template "metrics-custom.otel_test" with pattern "metrics-custom.otel_test.otel-*"

  @3.6.0
  Scenario: Integration OTel package with dataset variable uses default dataset
    Given the "dataset_integration_otel" package is installed
      And a policy is created with "dataset_integration_otel" package and "0.0.1" version
     Then the compiled policy has dataset "dataset_integration_otel.metrics" for "otelcol" input type
     # Index template not checked for integration packages: https://github.com/elastic/kibana/issues/160775
     # Then there is an index template "metrics-dataset_integration_otel.metrics" with pattern "metrics-dataset_integration_otel.metrics.otel-*"
