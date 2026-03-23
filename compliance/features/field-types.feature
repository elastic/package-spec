Feature: Field types
  Check support for field types that have been added since Fleet exists.

  @3.1.0
  Scenario: Package uses the "counted_keyword" type
   Given the "counted_keyword" package is installed
     And a policy is created with "counted_keyword" package
    Then index template "metrics-counted_keyword.foo" has a field "foo.count" with "type:counted_keyword"
