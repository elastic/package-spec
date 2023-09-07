include ../go/common.mk

build:
	GOOS=js GOARCH=wasm go build -o validator.wasm .

test:
	GOOS=js GOARCH=wasm go test -v -exec="`go env GOROOT`/misc/wasm/go_js_wasm_exec" . ../go/...
