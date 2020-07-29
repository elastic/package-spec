SPEC_DIR=resources/spec

# Updates the spec to the latest copy
update:
	cp -r ../../versions ${SPEC_DIR}

# Checks that the spec is up-to-date
check:
	diff -qr ../../versions ${SPEC_DIR}/versions
