var wasmBuffer;
fetch('validator.wasm').then((response) => {
  return response.arrayBuffer();
}).then((buffer) => {
  wasmBuffer = buffer;
})
const go = new Go();

function validateZipBuffer(file, buffer, success, error) {
	WebAssembly.instantiate(wasmBuffer, go.importObject).then((validator) => {
		go.run(validator.instance);

		elasticPackageSpec.validateFromZipReader(file, buffer).then(() => 
			success()
		).catch((err) => 
			error(err)
		).finally(() =>
			elasticPackageSpec.stop()
		)
	}).catch((err) => {
		console.error(err);
	});
}
