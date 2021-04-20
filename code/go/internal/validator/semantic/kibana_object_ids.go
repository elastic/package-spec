package semantic

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elastic/package-spec/code/go/internal/validator"

	"github.com/pkg/errors"

	"github.com/elastic/package-spec/code/go/internal/pkgpath"
)

func ValidateKibanaObjectIDs(pkgRoot string) error {
	filePaths := filepath.Join(pkgRoot, "kibana", "dashboard", "*.json")
	dashboardFiles, err := pkgpath.Files(filePaths)
	if err != nil {
		return errors.Wrap(err, "unable to find dashboard files")
	}

	var errs validator.ValidationErrors
	for _, dashboardFile := range dashboardFiles {
		name := dashboardFile.Name()
		dashboardID, err := dashboardFile.Values("$.id")
		if err != nil {
			return errors.Wrap(err, "unable to get dashboard ID")
		}

		fileID := strings.TrimRight(name, ".json")
		if fileID != dashboardID {
			err := fmt.Errorf("dashboard file '%s' defines non-matching ID '%s'", name, dashboardID)
			errs = append(errs, err)
		}
	}

	return errs
}
