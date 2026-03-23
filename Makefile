.PHONY: test

# Updates the spec in language libraries
update: code/* compliance
	@$(foreach lang,$^,set -e ; make -C $(lang) update;)

# Checks that language libraries have latest specs
check: code/* compliance
	@$(foreach lang,$^,set -e ; make -C $(lang) check;)

# Tests the language libraries' code
test: code/*
	@$(foreach lang,$^,set -e ; make -C $(lang) test;)

# Tests the language libraries' code to produce the required test files for the CI
test-ci: code/*
	@$(foreach lang,$^,set -e ; make -C $(lang) test-ci;)
