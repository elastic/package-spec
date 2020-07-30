package validator

type Package struct {
	SpecVersion string
	RootPath    string
}

func NewPackage(pkgRootPath string) (*Package, error) {
	// TODO: Check that package root path exists and is a folder
	// TODO: Check that package root path contains manifest file

	// Load package's manifest file and parse spec version
	specVersion := "1.0.0" // TODO: read from manifest

	// Instantiate Package object and return it
	p := Package{
		specVersion,
		pkgRootPath,
	}

	return &p, nil
}
