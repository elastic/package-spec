package validator

import (
	"github.com/elastic/package-spec/code/go/internal/validator"
)

// ValidateFromPath validates a package located at the given path against the
// appropriate specification and returns any errors.
func ValidateFromPath(packageRootPath string) error {
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
