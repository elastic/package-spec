#!/usr/bin/env node

if (process.argv.length < 3) {
        console.error("usage: " + process.argv[1] + " [package path]");
        process.exit(1);
}


require('./validate.lib.js');

const fs = require('fs');

const packagePath = process.argv[2];
const packageBuffer = fs.readFileSync(packagePath);

var result = elasticPackageSpec.validateFromBuffer(packagePath, packageBuffer);
console.log(result);
