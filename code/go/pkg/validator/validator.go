package validator

import (
	"github.com/elastic/package-spec/code/go/internal/validator"
)

// Validate validates a given package against the spec and returns any errors.
func Validate(packageRootPath string) error {
	p, err := validator.NewPackage(packageRootPath)
	if err != nil {
		return err
	}

	s, err := validator.NewSpec(p.SpecVersion)
	if err != nil {
		return err
	}

	return s.ValidatePackage(*p)
}