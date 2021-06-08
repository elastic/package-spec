SPEC_DIR=internal

update: tools
	# Update the spec to the latest copy
	@statik -m -f -src ../../versions -dest ${SPEC_DIR} -p spec
	@echo "Specs updated to latest."

	# Add license headers
	@go-licenser -license Elastic

check: lint check-license check-spec

# "yamlschema" directory has been excluded from linting, because it contains implementations of gojsonschema interfaces
# which are not compliant with linter rules. The golint tool doesn't support ignore comments.
lint: tools
	@go list ./... | grep -v yamlschema | xargs -n 1 golint -set_exit_status

check-license: tools
	@go-licenser -license Elastic -d

# Checks that the spec is up-to-date
check-spec: tools
	@go get github.com/rakyll/statik
	@statik -m -f -src ../../versions -dest temp -p spec
	@go-licenser -license Elastic
	@diff -qr ${SPEC_DIR}/spec temp/spec >/dev/null; \
		retval=$$?; \
		rm -rf temp; \
		if [ $$retval -eq 0 ]; then \
			echo "Specs are up to date!"; \
		else \
			echo "Specs are not up to date. Please run make update."; \
		fi; \
		exit $$retval

# Runs tests
test:
    # -count=1 is included to invalidate the test cache. This way, if you run "make test" multiple times
    # you will get fresh test results each time. For instance, changing the source of mocked packages
    # does not invalidate the cache so having the -count=1 to invalidate the test cache is useful.
	@go test -v ./... -count=1

tools:
	@go get golang.org/x/lint/golint
	@go get github.com/rakyll/statik
	@go get github.com/elastic/go-licenser
	@go mod tidy
