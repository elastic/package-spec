include ./common.mk

# Runs tests
test:
	# -count=1 is included to invalidate the test cache. This way, if you run "make test" multiple times
	# you will get fresh test results each time. For instance, changing the source of mocked packages
	# does not invalidate the cache so having the -count=1 to invalidate the test cache is useful.
	@go test -v ./... -count=1
