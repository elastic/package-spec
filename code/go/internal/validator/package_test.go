package validator

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPackageGood(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	pkgRootPath := path.Join(cwd, "test", "packages", "good")
	pkg, err := NewPackage(pkgRootPath)
	require.NoError(t, err)
	require.Equal(t, "1.0.4", pkg.SpecVersion)
	require.Equal(t, pkgRootPath, pkg.RootPath)
}

func TestNewPackageNonExistent(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	pkgRootPath := path.Join(cwd, "test", "packages", "nonexistent")
	pkg, err := NewPackage(pkgRootPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no package found at")
	require.Nil(t, pkg)
}

func TestNewPackageNoManifest(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	pkgRootPath := path.Join(cwd, "test", "packages", "no_manifest")
	pkg, err := NewPackage(pkgRootPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no package manifest file found at")
	require.Nil(t, pkg)
}

func TestNewPackageBadManifest(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	pkgRootPath := path.Join(cwd, "test", "packages", "bad_manifest")
	pkg, err := NewPackage(pkgRootPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not read specification version")
	require.Nil(t, pkg)
}
