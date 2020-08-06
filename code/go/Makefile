SPEC_DIR=internal

# Updates the spec to the latest copy
update:
	@statik -m -f -src ../../versions -dest ${SPEC_DIR} -p spec
	@echo "Specs updated to latest."

# Checks that the spec is up-to-date
check:
	@statik -m -f -src ../../versions -dest temp -p spec
	@diff -qr ${SPEC_DIR}/spec temp/spec >/dev/null && echo "Specs are up to date!" || echo "Specs are not up to date. Please run make update."
	@rm -rf temp

# Runs tests
test:
	@go test -v ./...
