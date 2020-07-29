# Introduction

This repository contains the specifications for Elastic Packages, as served up by the [Elastic Package Registry (EPM)](https://github.com/elastic/package-registry).

In the future it may also contain code for validating said specifications.

# Purpose

Please use this repository to discuss any changes to the specification, either my making issues or PRs to the specification.

# Specification Format

An Elastic Package specification is determined by the package's folder structure, folder names, the presence of certain files within these folders, and the structure of those files' contents.

Each specification format is versioned. You will find folders at the root of this repository for each active version of the specification format. 

Within each version folder, there must be a `spec.yml` file. This file is the entry point for the specification for a package's contents. It expresses the package's expected folder structure, folder names, and expected files within these folders. The specification is expressed using a schema similar to [JSON Schema](https://json-schema.org/), but with a couple of differences:
- The `type` field can be either `folder` or `file`,
- A new field field, `specRef` is introduced to (recursively) reference other specification files, and
- The specification is written as YAML for readability.
Expected package files, e.g. `manifest.yml` themselves have a structure to their contents. This is expressed in specification files using JSON schema (but written as YAML for readability).

Note that the specification files primarily defined the structure (syntax) of a package's contents. To a limited extent they may also define some semantics, e.g. enumeration values for certain fields. Richer semantics, however, will need to be expressed as validation code.