package validator

type Spec struct {
	version string
}

func NewSpec(version string) (*Spec, error) {
	// Find spec for given format version
	s := Spec{
		version,
	}

	return &s, nil
}

func (s Spec) ValidatePackage(pkg Package) ValidationErrors {
	// Stitch together specification YAML files into mega YAML specification
	// Convert mega YAML non-JSON schema parts to JSON schema equivalents
	// Convert mega YAML specification into mega JSON object (so we have a valid JSON schema)
	// Stitch together package contents into mega JSON object
	// Validate mega JSON object representing package against mega JSON schema
	// Parse validation errors and make them friendlier so they make sense in the context of packages
	// Perform additional non-trivial semantic validations
	// Return validation errors
	return nil
}
