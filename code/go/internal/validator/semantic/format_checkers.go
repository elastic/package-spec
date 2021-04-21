package semantic

import (
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
)

const (
	RelativePathFormat   = "relative-path"
	DataStreamNameFormat = "data-stream-name"
)

// relativePathChecker is responsible for checking presence of the file path
type relativePathChecker struct {
	currentPath string
}

// IsFormat method checks if the path exists.
func (r relativePathChecker) IsFormat(input interface{}) bool {
	asString, ok := input.(string)
	if !ok {
		return false
	}

	path := filepath.Join(r.currentPath, asString)
	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	return true
}

// LoadRelativePathFormatChecker loads the relative-path format checker into the
// json-schema validation library.
func LoadRelativePathFormatChecker(currentPath string) {
	gojsonschema.FormatCheckers.Add(RelativePathFormat, relativePathChecker{
		currentPath: currentPath,
	})
}

// UnloadRelativePathFormatChecker unloads the relative-path format checker from the
// json-schema validation library.
func UnloadRelativePathFormatChecker() {
	gojsonschema.FormatCheckers.Remove(RelativePathFormat)
}

// LoadDataStreamNameFormatChecker loads the data-stream-name format checker into the
// json-schema validation library.
func LoadDataStreamNameFormatChecker(currentPath string) {
	gojsonschema.FormatCheckers.Add(DataStreamNameFormat, relativePathChecker{
		currentPath: filepath.Join(currentPath, "data_stream"),
	})
}

// UnloadDataStreamNameFormatChecker unloads the data-stream-name format checker from the
// json-schema validation library.
func UnloadDataStreamNameFormatChecker() {
	gojsonschema.FormatCheckers.Remove(DataStreamNameFormat)
}
