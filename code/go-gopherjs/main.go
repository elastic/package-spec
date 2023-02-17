package main

import (
	"bytes"

	"github.com/gopherjs/gopherjs/js"

	"github.com/elastic/package-spec/v2/code/go/pkg/validator"
)

const moduleName = "elasticPackageSpec"

func main() {
	js.Global.Set(moduleName, make(map[string]interface{}))
	module := js.Global.Get(moduleName)
	module.Set("validateFromBuffer", func(name string, buffer []byte) string {
		reader := bytes.NewReader(buffer)
		err := validator.ValidateFromZipReader(name, reader, int64(len(buffer)))
		if err != nil {
			return err.Error()
		}
		return ""
	})
}
