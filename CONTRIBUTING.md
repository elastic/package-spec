# Contributing

## Folder Item spec

### Using predefined placeholders in filename patterns

There are predefined placeholders that can be used in filename patterns. Standard patterns are simple regular expressions
like `[a-z0-9]+\.json`. In some cases it might be useful to introduce a harder requirement (binding), e.g. a filename should
be prefixed with a package name (`{PACKAGE_NAME}+-.+\.json`).

Currently, the following placeholders are available:

* `{PACKAGE_NAME}` - name of the package