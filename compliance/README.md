# Compliance suite for Package Spec

This compliance suite tests the level of implementation of specific versions of
the Package Spec.

The compliance suite is defined in the `compliance` directory. Tests are defined
in Gherkin syntax (aka Cucumber), and executed using Godog. Features can be found
in the [`compliance/features`](https://github.com/elastic/package-spec/tree/main/compliance/features)
directory. Each scenario for each feature is tagged with the Package Spec version
that it requires to work.

Step definitions can be found in [`compliance_test.go`](https://github.com/jsoriano/package-spec/blob/compliance-suite/compliance/compliance_test.go).
Additional Go code is used to initialize Kibana and Elasticsearch clients. There
is also a wrapper for `elastic-package`, used for build and installation of
packages. 

Reference packages used on each scenario can be found in [`compliance/testdata/packages`](https://github.com/jsoriano/package-spec/tree/compliance-suite/compliance/testdata/packages).

## Running the suite

To run the suite, you need a target to evaluate. The target is configured with
environment variables as described below.

Once the target is configured, the compliance suite can be executed with:
```shell
TEST_SPEC_VERSION=2.9.0 make -C compliance/ test
```

Execution can be customized with environment variables:
* `TEST_SPEC_FEATURES`, with a comma-separated list of paths relative to
  `compliance` with feature files to execute.
* `TEST_SPEC_VERSION`, with the version of the Package Spec to check. Setting this
  variable is mandatory.

### Checking compliance of an Elastic Stack

You can configure the compliance suite to check an Elastic Stack using the
following environment variables:

* `ELASTICSEARCH_HOST`, address to the Elasticsearch host.
* `ELASTICSEARCH_USERNAME`, username in Elasticsearch, also used for Kibana.
* `ELASTICSEARCH_PASSWORD`, password for the given username.
* `KIBANA_HOST`, address to the Kibana host.
* `CA_CERT`, certificate of the CA used to sign the SSL certificates, if needed.

If the stack is managed by [`elastic-package stack`](https://github.com/elastic/elastic-package/#elastic-package-stack),
you can setup the environment variables with `elastic-package shellinit`:
```shell
eval $(elastic-package stack shellinit)
```
