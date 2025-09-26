SPEC_DIR=internal

CODE_COVERAGE_REPORT_FOLDER = $(PWD)/build/test-coverage
CODE_COVERAGE_REPORT_NAME_UNIT = $(CODE_COVERAGE_REPORT_FOLDER)/coverage-unit-report
TEST_RESULTS_FOLDER = $(PWD)/build/test-results

golicenser_cmd = go run github.com/elastic/go-licenser
golint_cmd = go run golang.org/x/lint/golint
gotestsum_cmd = go run gotest.tools/gotestsum
gocobertura_cmd = go run github.com/boumenot/gocover-cobertura
goimports_cmd = go run golang.org/x/tools/cmd/goimports

update:
	# Add license headers
	@$(golicenser_cmd) -license Elastic

check: lint check-license check-spec format check-git-clean

# "yamlschema" directory has been excluded from linting, because it contains implementations of gojsonschema interfaces
# which are not compliant with linter rules. The golint tool doesn't support ignore comments.
lint:
	@go list ./... | grep -v yamlschema | xargs -n 1 $(golint_cmd) -set_exit_status

format:
	@${goimports_cmd} -local github.com/elastic/package-spec/ -w .

check-git-clean:
	git update-index --really-refresh
	git diff-index --quiet HEAD

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
	# -count=1 is included to invalidate the test cache. This way, if you run "make test" multiple times
	# you will get fresh test results each time. For instance, changing the source of mocked packages
	# does not invalidate the cache so having the -count=1 to invalidate the test cache is useful.
	@go test -v ./... -count=1

test-ci: $(CODE_COVERAGE_REPORT_NAME_UNIT) $(TEST_RESULTS_FOLDER)
	@$(gotestsum_cmd) --junitfile "$(TEST_RESULTS_FOLDER)/TEST-unit.xml" -- -count=1 -coverprofile=$(CODE_COVERAGE_REPORT_NAME_UNIT).out ./...
	@$(gocobertura_cmd) < $(CODE_COVERAGE_REPORT_NAME_UNIT).out > $(CODE_COVERAGE_REPORT_NAME_UNIT).xml
