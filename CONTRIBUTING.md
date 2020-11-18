# Contributing

## Folder Item spec

### Using predefined placeholders in filename patterns

There are predefined placeholders that can be used in filename patterns. Standard patterns are simple regular expressions
like `[a-z0-9]+\.json`. In some cases it might be useful to introduce a harder requirement (binding), e.g. a filename should
be prefixed with a package name (`{PACKAGE_NAME}-.+\.json`).

Currently, the following placeholders are available:

* `{PACKAGE_NAME}` - name of the package

## Folder Item schema

### Defining property format

The JSON Schema defines the basic structure of a JSON document (e.g. package manifests, ingest pipelines, etc.).
In some cases this might be insufficient as there are properties that require strict validation (not just type
consistency), e.g. format validation:

```yaml
src:
  description: Relative path to the screenshot's image file.
  type: string
  format: relative-path
  examples:
  - "/img/apache_httpd_server_status.png"
```

Currently, the following custom formats are available:

* `relative-path`: Relative path to the package root directory. The format checker verifies if the path is correct and
  the file exists.