package jsonschema

import (
	"os"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/loader"
)

func TestMarshal_AllJSSchema(t *testing.T) {
	// given
	cases := []struct {
		title               string
		version             string
		specPath            string
		filePath            string
		expectedError       bool
		expectedOutputPath  string
		expectedNumberFiles int
	}{
		{
			title:               "manifest from version 1.0.0",
			version:             "1.0.0",
			specPath:            "testdata/simple-spec",
			filePath:            "manifest.yml",
			expectedError:       false,
			expectedOutputPath:  "testdata/manifest-1.0.0.yml",
			expectedNumberFiles: 4,
		},
		{
			title:               "manifest from version 2.1.0",
			version:             "2.1.0",
			specPath:            "testdata/simple-spec",
			filePath:            "manifest.yml",
			expectedError:       false,
			expectedOutputPath:  "testdata/manifest-2.1.0.yml",
			expectedNumberFiles: 4,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			// given
			specVersion, err := semver.NewVersion(c.version)
			require.NoError(t, err)

			rootSpec, err := loader.LoadSpec(fspath.DirFS(c.specPath), *specVersion, "")
			require.NoError(t, err)

			// when
			rendered, err := AllJSONSchemas(rootSpec)

			// then
			if c.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			contents, err := os.ReadFile(c.expectedOutputPath)
			require.NoError(t, err)

			// check contents of one file
			for _, jsonschema := range rendered {
				if jsonschema.Name != c.filePath {
					continue
				}
				assert.Equal(t, string(contents), string(jsonschema.JSONSchema))
				break
			}

			assert.Equal(t, c.expectedNumberFiles, len(rendered))
		})
	}
}

func TestMarshal_GivenJSONSchema(t *testing.T) {
	// given
	cases := []struct {
		title              string
		version            string
		specPath           string
		filePath           string
		expectedError      bool
		expectedOutputPath string
	}{
		{
			title:              "not found",
			version:            "1.0.0",
			specPath:           "testdata/simple-spec",
			filePath:           "noexit.yml",
			expectedError:      true,
			expectedOutputPath: "",
		},
		{
			title:              "manifest from version 1.0.0",
			version:            "1.0.0",
			specPath:           "testdata/simple-spec",
			filePath:           "manifest.yml",
			expectedError:      false,
			expectedOutputPath: "testdata/manifest-1.0.0.yml",
		},
		{
			title:              "manifest from version 2.1.0",
			version:            "2.1.0",
			specPath:           "testdata/simple-spec",
			filePath:           "manifest.yml",
			expectedError:      false,
			expectedOutputPath: "testdata/manifest-2.1.0.yml",
		},
		{
			title:              "file with regex",
			version:            "1.0.0",
			specPath:           "testdata/simple-spec",
			filePath:           "data_2.yml",
			expectedError:      false,
			expectedOutputPath: "testdata/data-1.0.0.yml",
		},
		{
			title:              "file with regex not found",
			version:            "1.0.0",
			specPath:           "testdata/simple-spec",
			filePath:           "data_ng.yml",
			expectedError:      true,
			expectedOutputPath: "",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			// given
			specVersion, err := semver.NewVersion(c.version)
			require.NoError(t, err)

			rootSpec, err := loader.LoadSpec(fspath.DirFS(c.specPath), *specVersion, "")
			require.NoError(t, err)

			// when
			rendered, err := JSONSchema(rootSpec, c.filePath)

			// then
			if c.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			contents, err := os.ReadFile(c.expectedOutputPath)
			require.NoError(t, err)
			assert.Equal(t, string(contents), string(rendered.JSONSchema))
		})
	}
}
