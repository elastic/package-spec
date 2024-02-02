Feature: Subobjects
  Support to disable subobjects in object fields.

  @3.1.0
  Scenario: Installer leverages subobjects false
   Given the "subobjects_false" package is installed
     And a policy is created with "subobjects_false" package
    Then index template "metrics-subobjects_false.foo" has a field "foo.object" with "subobjects:false"
