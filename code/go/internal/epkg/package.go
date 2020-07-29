package epkg

type Package struct {
	FormatVersion string `yaml:format_version`
}

func NewFromPath(pkgRootPath string) (*Package, error) {
	// Load package's manifest file and parse version
	// Instantiate Package object and return it
}

func (p Package) Validate() ValidationErrors {
	return nil
}