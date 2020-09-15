SPEC_DIR=internal

# Updates the spec to the latest copy
update:
	@go get github.com/rakyll/statik
	@statik -m -f -src ../../versions -dest ${SPEC_DIR} -p spec
	@echo "Specs updated to latest."

check: lint check-spec 

# "yamlschema" directory has been excluded from linting, because it contains implementations of gojsonschema interfaces
# which are not compliant with linter rules. The golint tool doesn't support ignore comments.
lint:
	@go get -u golang.org/x/lint/golint
	@go list ./... | grep -v yamlschema | xargs -n 1 golint -set_exit_status

# Checks that the spec is up-to-date
check-spec:
	@go get github.com/rakyll/statik
	@statik -m -f -src ../../versions -dest temp -p spec
	@diff -qr ${SPEC_DIR}/spec temp/spec >/dev/null && echo "Specs are up to date!" || echo "Specs are not up to date. Please run make update."
	@rm -rf temp

# Runs tests
test:
    # -count=1 is included to invalidate the test cache. This way, if you run "make test" multiple times
    # you will get fresh test results each time. For instance, changing the source of mocked packages
    # does not invalidate the cache so having the -count=1 to invalidate the test cache is useful.
	@go test -v ./... -count=1
