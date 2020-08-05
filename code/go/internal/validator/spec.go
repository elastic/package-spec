package validator

import (
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
)

type Spec struct {
	version  semver.Version
	specPath string
}

func NewSpec(version semver.Version) (*Spec, error) {
	majorVersion := strconv.FormatUint(version.Major(), 10)
	specPath := path.Join("..", "..", "resources", "spec", "versions", majorVersion)
	info, err := os.Stat(specPath)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "no specification found for version [%v]", version)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("no valid specification found for version [%v]", version)
	}

	s := Spec{
		version,
		specPath,
	}

	return &s, nil
}

func (s Spec) ValidatePackage(pkg Package) ValidationErrors {
	var errs ValidationErrors

	rootSpecPath := path.Join(s.specPath, "spec.yml")
	rootSpec, err := newFolderSpec(rootSpecPath)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "could not read root folder spec file"))
		return errs
	}

	return rootSpec.Validate(pkg.RootPath)
}
