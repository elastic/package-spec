# Introduction

This repository contains:
* specifications for Elastic Packages, as served up by the [Elastic Package Registry (EPR)](https://github.com/elastic/package-registry). There may be multiple versions of the specifications; these are resolved when loading the spec depending on the `format_version` of the package. Read more in the _Specification Versioning_ section below.
* code libraries for validating said specifications; these can be found under the `code` top-level folder.

Please use this repository to discuss any changes to the specification, either my making issues or PRs to the specification.

# What is an Elastic Package?

An Elastic Package is a collection of assets for the Elastic Stack. In addition, it contains manifest files which contain additional information about the package. The exact content and structure of a package are described by the package spec.

A package with all its assets is downloaded as a .zip file from the package-registry by Fleet inside Kibana. The assets are then unpacked and each asset is installed into the related API and the package can be configured.

In the following is a high level overview of a package.

## Asset organisation

In general, assets within a package are organised by {stack-component}/{asset-type}. For example assets for Elasticsearch ingest pipelines are in the folder `elasticsearch/ingest-pipeline`. The same logic applies to all Elasticsearch, Kibana and Elastic Agent assets.

There is a special folder `data_stream`. All assets inside the `data_stream` folder must follow the [Data Stream naming scheme](https://www.elastic.co/blog/an-introduction-to-the-elastic-data-stream-naming-scheme). `data_stream` can contain multiple folders, each with the name of that describes the content. Inside this folder, the same structure as before for {stack-component}/{asset-type} applies. The difference is that for all these assets, Fleet during installation enforces naming rules related to the [Data Stream naming scheme](https://www.elastic.co/blog/an-introduction-to-the-elastic-data-stream-naming-scheme). All assets in this folder belong directly or indirectly to data streams.

In contrast, any asset added on the top level will be picked up as json document, pushed to the corresponding Elasticsearch / Kibana APIs and used as is. In most scenarios, only data stream assets are needed. There are exceptions where global assets are required to get more flexibility. This could be, for example, an ILM policy that applies to all data streams.

## Supported assets

For a quick overview, these are the assets typically found in an Elastic Package. The package spec will always contain the fully up-to-date list.

* Elasticsearch
  * Ingest Pipeline
  * Index Template
  * Transform
  * Index template settings
* Kibana
  * Dashboards
  * Visualization
  * Index patterns
  * ML Modules
  * Map
  * Search
  * Security rules
  * CSP (cloud security posture) rule templates
* Other
  * fields.yml

The special asset `fields.yml` is used to generate out of a single definition Elasticsearch Index Templates and Kibana index patterns. The exact definition can be found in the package spec.


# Specification Format

An Elastic Package specification describes:
1. the folder structure of packages and expected files within these folders; and
2. the structure of the expected files' contents.

In the spec folder there is be a `spec.yml` file. This file is the entry point for the
 specification for a package's contents. It describes the folder structure of packages and expected
files within these folders (this is point 1. above). The specification is expressed using a schema similar
to [JSON Schema](https://json-schema.org/), but with a couple of differences:
-- The `type` field can be either `folder` or `file`,
-- A new field, `contents` is introduced to (recursively) describe the contents of folders (i.e. when ty
pe == folder), and
-- The specification is written as YAML for readability.

Expected package files, e.g. `manifest.yml` themselves have a structure to their contents. This structure is described in specification files using JSON schema (this is point 2. above). These specification files are also written as YAML for readability.

Note that the specification files primarily define the structure (syntax) of a package's contents. To a limited extent they may also define some semantics, e.g. enumeration values for certain fields. Richer semantics, however, will need to be expressed as validation code.

# Specification Versioning

Package specifications are versioned. Versions follow the [semantic versioning](https://semver.org/) scheme. In the context of package specifications, this means the following.

* Packages must specify the specification version they are using. This is done via the `format_version` property in the package's root `manifest.yml` file. The value of `format_version` must conform to the semantic versioning scheme.

* Specifications are defined by schema files and semantic rules, some attributes or files will only be available since, or till a version.

* Note that the latest version of each major may include a pre-release suffix, `e.g. 1.4.0-alpha1`. This indicates that this version is still under development and may be changed multiple times. Once the pre-relase suffix is removed, however, the specification at that version becomes immutable. Further changes must follow the process outlined below in _Changing a Specification_.

## Changing a Specification

* Consider a proposal to change the specification in some way. The version number of the changed specification must be determined as follows:
  * If the proposed change makes the specification stricter than it is at `x.y.z`, the new version number will be `X.0.0`, where `X = x + 1`. That is, we bump up the major version. 
     * If the change is in a schema file, consider the `spec` the latest
       version, and add a JSON patch in the `versions` section to support older
       schemas.
     * If the change is in semantic rules, add a constraint in the rule, so they only apply on
       the indicated version range.
     * Add a changelog entry in the `spec/changelog.yml` file in the section of this major.
  * If the proposed change makes the specification looser than it is at `x.y.z`, the new version number will be `x.Y.0`, where `Y = y + 1`. That is, we bump up the minor version and create a new changelog section in the `spec/changelog.yml` file. Note that adding new, but optional, constraints to a specification is a change that makes a specification looser.
  * If the proposed change does not change the strictness of the specification at `x.y.z`, the new version number will be `x.y.Z`, where `Z = z + 1`. That is, we bump the patch version.
     * Apply the proposed changes to the existing specification under the `spec` folder.
     * Set the root-level `version` property in the specification's root `spec.yml` file to `x.y.Z`.
     * Add a changelog entry in the `spec/changelog.yml` file in the section for
       this version.

## Version Compatibility between Packages and Specifications

A package specifying its `format_version` as `x.y.z` must be valid against specifications in the semantic version range `[x.y.z, X.0.0)`, where `X = x + 1`.
