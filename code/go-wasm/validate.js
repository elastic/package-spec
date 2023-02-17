#!/usr/bin/env node

if (process.argv.length < 3) {
	console.error("usage: " + process.argv[1] + " [package path]");
	process.exit(1);
}

// From wasm_exec_node.js
globalThis.require = require;
globalThis.fs = require("fs");
globalThis.TextEncoder = require("util").TextEncoder;
globalThis.TextDecoder = require("util").TextDecoder;

globalThis.performance = {
	now() {
		const [sec, nsec] = process.hrtime();
		return sec * 1000 + nsec / 1000000;
	},
};

const crypto = require("crypto");
globalThis.crypto = {
	getRandomValues(b) {
		crypto.randomFillSync(b);
	},
};

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
