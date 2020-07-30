package validator

type Package struct {
	SpecVersion string
	RootPath string
}

func NewPackage(pkgRootPath string) (*Package, error) {
	// Load package's manifest file and parse spec version
	specVersion := "1.0.0"

	// Instantiate Package object and return it
	p := Package{
		specVersion,
		pkgRootPath,
	}

	return &p, nil
}