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
	return nil
}