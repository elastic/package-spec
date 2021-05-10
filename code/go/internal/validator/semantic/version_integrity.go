package semantic

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/pkgpath"
)

// ValidateVersionIntegrity returns validation errors if the version defined in manifest isn't referenced in the latest
// entry of the changelog file.
func ValidateVersionIntegrity(pkgRoot string) ve.ValidationErrors {
	manifestVersion, err := readManifestVersion(pkgRoot)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	latestChangelogVersion, err := readLatestChangelogVersion(pkgRoot)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	if manifestVersion != latestChangelogVersion {
		return ve.ValidationErrors{fmt.Errorf("inconsistent versions between manifest (%s) and changelog (%s)",
			manifestVersion, latestChangelogVersion)}
	}
	return nil
}

func readManifestVersion(pkgRoot string) (string, error) {
	manifestPath := filepath.Join(pkgRoot, "manifest.yml")
	f, err := pkgpath.Files(manifestPath)
	if err != nil {
		return "", errors.Wrap(err, "can't locate manifest file")
	}

	if len(f) != 1 {
		return "", errors.New("single manifest file expected")
	}

	val, err := f[0].Values("version")
	if err != nil {
		return "", errors.Wrap(err, "can't read manifest version")
	}
	return val.(string), nil
}

func readLatestChangelogVersion(pkgRoot string) (string, error) {
	var manifest []struct {
		Version string `yaml:"version"`
	}

	body, err := ioutil.ReadFile(filepath.Join(pkgRoot, "changelog.yml"))
	if err != nil {
		return "", errors.Wrap(err, "can't read changelog file")
	}

	err = yaml.Unmarshal(body, &manifest)
	if err != nil {
		return "", errors.Wrap(err, "can't unmarshal changelog file")
	}

	if len(manifest) == 0 {
		return "", errors.Wrap(err, "changelog file is empty")
	}
	return manifest[0].Version, nil
}
