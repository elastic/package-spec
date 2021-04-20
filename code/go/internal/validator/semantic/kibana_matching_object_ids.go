package semantic

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elastic/package-spec/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/code/go/internal/validator"

	"github.com/pkg/errors"
)

// ValidateKibanaObjectIDs returns validation errors if there are any Kibana
// object files that define IDs not matching the file's name. That is, it returns
// validation errors if a Kibana object file, foo.json, in the package defines
// an object ID other than foo inside it.
func ValidateKibanaObjectIDs(pkgRoot string) error {
	filePaths := filepath.Join(pkgRoot, "kibana", "*", "*.json")
	objectFiles, err := pkgpath.Files(filePaths)
	if err != nil {
		return errors.Wrap(err, "unable to find Kibana object files")
	}

	var errs validator.ValidationErrors
	for _, objectFile := range objectFiles {
		name := objectFile.Name()
		objectID, err := objectFile.Values("$.id")
		if err != nil {
			return errors.Wrap(err, "unable to get Kibana object ID")
		}

		fileID := strings.TrimRight(name, ".json")
		if fileID != objectID {
			err := fmt.Errorf("kibana object file '%s' defines non-matching ID '%s'", name, objectID)
			errs = append(errs, err)
		}
	}

	return errs
}
