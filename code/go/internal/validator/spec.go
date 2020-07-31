package validator

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
)

type Spec struct {
	version string
}

func NewSpec(version string) (*Spec, error) {
	specPath := path.Join("..", "..", "resources", "spec", "versions", version)
	info, err := os.Stat(specPath)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "no specification found for version [%v]", version)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("no valid specification found for version [%v]", version)
	}

	s := Spec{
		version,
	}

	return &s, nil
}

func (s Spec) ValidatePackage(pkg Package) ValidationErrors {
	var errs ValidationErrors

	specJsonSchema, err := s.toJsonSchema()
	if err != nil {
		errs = append(errs, errors.Wrap(err, "could not convert specification to JSON schema"))
		return errs
	}

	packageJson, err := pkg.ToJson()
	if err != nil {
		errs = append(errs, errors.Wrap(err, "could not convert package contents to JSON"))
		return errs
	}

	// Validate mega JSON object representing package against mega JSON schema
	schemaLoader := gojsonschema.NewStringLoader(specJsonSchema)
	documentLoader := gojsonschema.NewStringLoader(packageJson)
	validationResult, err := gojsonschema.Validate(schemaLoader, documentLoader)

	// Parse validation errors and make them friendlier so they make sense in the context of packages
	if !validationResult.Valid() {
		for _, err := range validationResult.Errors() {
			// TODO: translate to friendlier errors before appending
			errs = append(errs, errors.New(err.String()))
		}

		return errs
	}

	// TODO: Perform additional non-trivial semantic validations

	// Return validation errors
	return errs
}

func (s Spec) toJsonSchema() (string, error) {
	// Stitch together specification YAML files into mega YAML specification
	// Convert mega YAML non-JSON schema parts to JSON schema equivalents
	// Convert mega YAML specification into mega JSON object (so we have a valid JSON schema)
	return "", nil
}
