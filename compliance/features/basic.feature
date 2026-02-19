Feature: Basic package types support
  Basic tests with minimal packages

  @1.0.0
  Scenario: Integration package can be installed
   Given the "basic_integration" package is installed
     And a policy is created with "basic_integration" package
    Then there is an index template "metrics-basic_integration.foo" with pattern "metrics-basic_integration.foo-*"

  @2.6.0
  Scenario: Input package can be installed
   Given the "basic_input" package is installed
     And a policy is created with "basic_input" package, "1.0.0" version, "test" template, "test" input, "logfile" input type and dataset "spec.input_test"
    Then there is an index template "logs-spec.input_test" with pattern "logs-spec.input_test-*"

  @3.5.0
  Scenario: OTel input package can be installed
   Given the "good_input_otel" package is installed
     And a policy is created with "good_input_otel" package, "0.0.1" version, "httpcheckreceiver" template, "httpcheckreceiver" input, "otelcol" input type and dataset "spec.otel_input_test"
    Then there is an index template "metrics-spec.otel_input_test" with pattern "metrics-spec.otel_input_test.otel-*"

  @3.3.0
  Scenario: Basic content package can be installed
   Given the "basic_content" package is installed
     And prebuilt detection rules are loaded
    Then there is a dashboard "basic_content-dashboard-abc-1"
     And there is a detection rule "12cea9e9-5766-474d-a9dc-34ef7c7677c7"

  @3.6.0
  Scenario: Content package can be installed
   Given the "good_content" package is installed
     And prebuilt detection rules are loaded
    Then there is a dashboard "good_content-dashboard-abc-1"
     # Missing support in Kibana (Fleet) https://github.com/elastic/kibana/pull/186974
     And there is an SLO "good_content-slo-abc-1"
     And there is a detection rule "12cea9e9-5766-474d-a9dc-34ef7c7677c6"

  @3.6.0
  Scenario: Integration package with OTel input can be installed
   Given the "good_v3" package is installed
     And a policy is created with "good_v3" package, "1.1.0" version, "otel" template, "otelcol" input type and dataset "good_v3.otel_test"
    Then there is an index template "logs-good_v3.otel_test" with pattern "logs-good_v3.otel_test.otel-*"
  
  @3.6.0
  Scenario: OTel input package with profiles type can be installed
   Given the "good_input_profiles" package is installed
     And a policy is created with "good_input_profiles" package, "1.0.0" version, "profilingreceiver" template, "profilingreceiver" input, "otelcol" input type and dataset "spec.otel_input_test"
    Then there is an index template "profiles-spec.otel_input_test" with pattern "profiles-spec.otel_input_test.otel-*"
