LIB_GOLANG_DIR=code/go
LIB_GOLANG_SPEC_DIR=${LIB_GOLANG_DIR}/resources/spec

# Updates the spec in language libraries
update:
	cp -r versions $LIB_GOLANG_SPEC_DIR

# Checks that language libraries have latest specs
check:
	diff -qr versions ${LIB_GOLANG_SPEC_DIR}/versions
