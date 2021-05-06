package semantic

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
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
	var manifest = struct {
		Version string `yaml:"version"`
	}{}

	body, err := ioutil.ReadFile(filepath.Join(pkgRoot, "manifest.yml"))
	if err != nil {
		return "", errors.Wrap(err, "can't read manifest file")
	}

	err = yaml.Unmarshal(body, &manifest)
	if err != nil {
		return "", errors.Wrap(err, "can't unmarshal manifest file")
	}
	return manifest.Version, nil
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
