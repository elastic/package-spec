#!/usr/bin/env node

if (process.argv.length < 3) {
	console.error("usage: " + process.argv[1] + " [package path]");
	process.exit(1);
}

const path = require('path')
const fs = require('fs');

require(path.join(process.env.GOROOT, "misc/wasm/wasm_exec.js"));
const go = new Go();

const wasmBuffer = fs.readFileSync('validator.wasm');
WebAssembly.instantiate(wasmBuffer, go.importObject).then((validator) => {
	go.run(validator.instance);

	elasticPackageSpec.validateFromZip(process.argv[2]).then(() =>
		console.log("OK")
	).catch((err) =>
		console.error(err)
	).finally(() =>
		elasticPackageSpec.stop()
	)
}).catch((err) => {
	console.error(err);
});
