package validator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/pkg/errors"
)

type Package struct {
	SpecVersion string
	RootPath    string
}

func NewPackage(pkgRootPath string) (*Package, error) {
	info, err := os.Stat(pkgRootPath)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "no package found at path [%v]", pkgRootPath)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("no package folder found at path [%v]", pkgRootPath)
	}

	pkgManifestPath := path.Join(pkgRootPath, "manifest.yml")
	info, err = os.Stat(pkgManifestPath)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "no package manifest file found at path [%v]", pkgManifestPath)
	}

	data, err := ioutil.ReadFile(pkgManifestPath)
	if err != nil {
		return nil, fmt.Errorf("could not read package manifest file [%v]", pkgManifestPath)
	}

	var manifest struct {
		SpecVersion string `yaml:"format_version"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, errors.Wrapf(err, "could not parse package manifest file [%v]", pkgManifestPath)
	}

	if manifest.SpecVersion == "" {
		return nil, fmt.Errorf("could not read specification version from package manifest file [%v]", pkgManifestPath)
	}

	// Instantiate Package object and return it
	p := Package{
		manifest.SpecVersion,
		pkgRootPath,
	}

	return &p, nil
}

func (p Package) ToJson() (string, error) {
	// Stitch together package contents into mega JSON object
	return "", nil
}
