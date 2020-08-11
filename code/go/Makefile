SPEC_DIR=internal

# Updates the spec to the latest copy
update:
	@go get github.com/rakyll/statik
	@statik -m -f -src ../../versions -dest ${SPEC_DIR} -p spec
	@echo "Specs updated to latest."

check: lint check-spec 

lint:
	@go get -u golang.org/x/lint/golint
	@go list ./... | xargs -n 1 golint -set_exit_status

# Checks that the spec is up-to-date
check-spec:
	@go get github.com/rakyll/statik
	@statik -m -f -src ../../versions -dest temp -p spec
	@diff -qr ${SPEC_DIR}/spec temp/spec >/dev/null && echo "Specs are up to date!" || echo "Specs are not up to date. Please run make update."
	@rm -rf temp

# Runs tests
test:
	@go test -v ./...
