## CLI

* Build wasm with `make build`.
* Run `./validate.js /path/to/package.zip` (or `node validate.js /path/to/package.zip`).

Node.js and Go are required.

## Web server with client validation

* Build wasm with `make build`.
* Run the server with `go run server.go`.
* Access http://localhost:8080 and select a file.
