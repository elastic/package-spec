SPEC_DIR=resources/spec

# Updates the spec to the latest copy
update:
	cp -r ../../versions ${SPEC_DIR}

# Checks that the spec is up-to-date
check:
	diff -qr ../../versions ${SPEC_DIR}/versions

# Builds the go module and installs it
build:
	go get github.com/elastic/package-spec/code/go

# Runs tests
test:
	go test -v ./...
