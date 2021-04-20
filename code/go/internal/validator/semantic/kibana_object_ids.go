package semantic

import (
	"path/filepath"

	"github.com/elastic/package-spec/code/go/internal/pkgpath"
)

func ValidateKibanaObjectIDs(pkgRoot string) error {
	filePaths := filepath.Join(pkgRoot, "kibana", "dashboard", "*.json")
	_, err := pkgpath.Files(filePaths)
	if err != nil {
		return err
	}

	return nil
}
