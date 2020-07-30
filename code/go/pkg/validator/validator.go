package validator

import (
	"github.com/elastic/package-spec/code/go/internal/validator"
)

// ValidateFromPath validates a package located at the given path against the
// appropriate specification and returns any errors.
func ValidateFromPath(packageRootPath string) error {
	pkg, err := validator.NewPackage(packageRootPath)
	if err != nil {
		return err
	}

	spec, err := validator.NewSpec(pkg.SpecVersion)
	if err != nil {
		return err
	}

	return spec.ValidatePackage(*pkg)
}
