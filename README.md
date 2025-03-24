# Introduction

This repository contains:
* Specifications for Elastic Packages, as served up by the [Elastic Package Registry (EPR)](https://github.com/elastic/package-registry). There may be multiple versions of the specifications; these are resolved when loading the spec depending on the `format_version` of the package. Read more in the _Specification Versioning_ section below.
* Code libraries for validating said specifications; these can be found under the `code` top-level folder.

Please use this repository to discuss any changes to the specification, either by making issues or PRs to the specification.

# What is an Elastic Package?

An Elastic Package is a collection of assets for the Elastic Stack. In addition, it contains manifest files which contain additional information about the package. The exact content and structure of a package are described by the Package Spec.

A package with all its assets is downloaded as a .zip file from the package-registry by Fleet inside Kibana. The assets are then unpacked and each asset is installed into the related API and the package can be configured.

In the following is a high level overview of a package.

## Asset organisation

In general, assets within a package are organised by {stack-component}/{asset-type}. For example assets for Elasticsearch ingest pipelines are in the folder `elasticsearch/ingest-pipeline`. The same logic applies to all Elasticsearch, Kibana and Elastic Agent assets.

There is a special folder `data_stream`. All assets inside the `data_stream` folder must follow the [Data Stream naming scheme](https://www.elastic.co/blog/an-introduction-to-the-elastic-data-stream-naming-scheme). `data_stream` can contain multiple folders, each with the name of that describes the content. Inside this folder, the same structure as before for {stack-component}/{asset-type} applies. The difference is that for all these assets, Fleet during installation enforces naming rules related to the [Data Stream naming scheme](https://www.elastic.co/blog/an-introduction-to-the-elastic-data-stream-naming-scheme). All assets in this folder belong directly or indirectly to data streams.

In contrast, any asset added on the top level will be picked up as json document, pushed to the corresponding Elasticsearch / Kibana APIs and used as is. In most scenarios, only data stream assets are needed. There are exceptions where global assets are required to get more flexibility. This could be, for example, an ILM policy that applies to all data streams.

## Supported assets

For a quick overview, these are the assets typically found in an Elastic Package. The Package Spec will always contain the fully up-to-date list.

* Elasticsearch
  * Ingest Pipeline
  * Index Template
  * Transform
  * Index template settings
* Kibana
  * Dashboards
  * Visualization
  * Index patterns
  * Lens
  * ML Modules
  * Map
  * Search
  * Security rules
  * CSP (cloud security posture) rule templates
  * SLOs
  * Osquery pack assets.
  * Osquery saved queries.
  * Tags
* Other
  * fields.yml

The special asset `fields.yml` is used to generate out of a single definition Elasticsearch Index Templates and Kibana index patterns. The exact definition can be found in the Package Spec.


# Specification Format

An Elastic Package specification describes:
1. the folder structure of packages and expected files within these folders; and
2. the structure of the expected files' contents.

In the spec folder there is be a `spec.yml` file. This file is the entry point for the
 specification for a package's contents. It describes the folder structure of packages and expected
files within these folders (this is point 1. above). The specification is expressed using a schema similar
to [JSON Schema](https://json-schema.org/), but with a couple of differences:
- The `type` field can be either `folder` or `file`,
- A new field, `contents` is introduced to (recursively) describe the contents of folders (i.e. when type == folder), and
- The specification is written as YAML for readability.

Expected package files, e.g. `manifest.yml` themselves have a structure to their contents. This structure is described in specification files using JSON schema (this is point 2. above). These specification files are also written as YAML for readability.

Note that the specification files primarily define the structure (syntax) of a package's contents. To a limited extent they may also define some semantics, e.g. enumeration values for certain fields. Richer semantics, however, will need to be expressed as validation code.

# Specification Versioning

Package Spec version follows [semantic versioning](https://semver.org) for its
compatibility with the Stack and only partially for the packages format.
That means that patch versions may include stricter validations for packages,
but they should not include support for new features.
Major versions are reserved for significant changes in the format of the files,
the structure of packages or the interpretation of the Package Spec.

Packages must specify the specification version they are using. This is done via
the `format_version` property in the package's root `manifest.yml` file. The value
of `format_version` must conform to the semantic versioning scheme.

Specifications are defined by schema files and semantic rules, some attributes or
files will only be available since, or till a version.

Note that some versions may include a pre-release suffix, `e.g. 1.4.0-alpha1`. This
indicates that these versions are still under development and may be changed multiple
times. These versions in development can be used in pre-release versions of
packages, but breaking changes can still occur.
Once the pre-release suffix is removed, however, the specification at that version becomes
immutable. Further changes must follow the process outlined below in _Changing a Specification_.

## Changing a Specification

Consider a proposal to change the specification in some way. The version number
of the changed specification must be determined as follows:

  * If the proposed change modifies the format of the files in a way that
    require manual adjustments in packages, the new version number will be `X.0.0`,
    where `X = x + 1`. That is, we bump up the major version.
    There are some exceptions, for changes that could be done in patch versions:
    * When the proposed change is intended to address existing issues
      in packages like ambiguous mappings or security risks.
    * When the proposed change affects a feature marked as technical preview.
  * If the proposed change introduces support for a new feature that requires
    explicit support in the Stack, the new version will be `x.Y.0`, where
    `Y = y + 1`. That is, we bump up the minor version. See note below about
    compatibility between packages and the Stack.
  * Any other change would be included in the next patch version, `x.y.Z` where
    `Z = z + 1`. This includes any change on validation that doesn't neccesarily
    lead to a change in the behaviour of the installed package.

If the change is in a schema file, add a JSON patch in the `versions` section to
continue supporting the previous format.

If the change is in semantic rules, add a constraint in the rule, so they only
apply on the indicated version range and package types.

Remember to add a changelog entry in `spec/changelog.yml` for any change in the
spec. If no section exists for the version determined by the above rules, please
add the new section. Multiple `next` versions may exist at the same moment if
multiple versions are in development.

## Version Compatibility between Packages and Specifications

A package specifying its `format_version` as `x.y.z` must be valid against specifications in the semantic version range `[x.y.z, X.0.0)`, where `X = x + 1`.

## Version Compatibility between Packages and the Stack

Starting on Package Spec v3 and for some Elastic offerings, compatibility
between packages and the Stack is based on the major and minor Package Spec
version.

Eventually all Elastic Stack offerings will have ranges of compatible versions.
In these ranges the patch is ignored. So for example a Stack could be declared
compatible with a minimum spec version of `2.0` and maximum of `3.0`. This would
mean that it is compatible with packages using any spec version >= 2.0.0 and <3.1.0.

## Contributing

Please check out our [contributing documentation](./CONTRIBUTING.md) for guidelines about how to contribute in the specification for Elastic Packages.
