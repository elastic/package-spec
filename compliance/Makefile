CODE_COVERAGE_REPORT_FOLDER = $(PWD)/build/test-coverage
CODE_COVERAGE_REPORT_NAME_UNIT = $(CODE_COVERAGE_REPORT_FOLDER)/coverage-unit-report
TEST_RESULTS_FOLDER = $(PWD)/build/test-results

golicenser_cmd = go run github.com/elastic/go-licenser
golint_cmd = go run golang.org/x/lint/golint
gotestsum_cmd = go run gotest.tools/gotestsum

update:
	# Add license headers
	@$(golicenser_cmd) -license Elastic

check: lint check-license check-spec

# "yamlschema" directory has been excluded from linting, because it contains implementations of gojsonschema interfaces
# which are not compliant with linter rules. The golint tool doesn't support ignore comments.
lint:
	@go list ./... | grep -v yamlschema | xargs -n 1 $(golint_cmd) -set_exit_status

check-license:
	@$(golicenser_cmd) -license Elastic -d

# Checks that the spec is up-to-date
check-spec:
	@$(golicenser_cmd) -license Elastic

$(CODE_COVERAGE_REPORT_NAME_UNIT):
	mkdir -p $@

$(TEST_RESULTS_FOLDER):
	mkdir -p $@

# Runs tests
test:
	@go test -v ./...

test-ci: $(CODE_COVERAGE_REPORT_NAME_UNIT) $(TEST_RESULTS_FOLDER)
	@$(gotestsum_cmd) --junitfile "$(TEST_RESULTS_FOLDER)/TEST-unit.xml" -- -count=1 ./...
