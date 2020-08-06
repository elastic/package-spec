package spec

import (
	"testing"

	"github.com/rakyll/statik/fs"
	"github.com/stretchr/testify/require"
)

func TestBundledSpecs(t *testing.T) {
	bundle, err := fs.New()
	require.NoError(t, err)

	_, err = bundle.Open("/1/spec.yml")
	require.NoError(t, err)
}
