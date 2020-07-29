# Introduction

This repository contains the specifications for Elastic Packages, as served up by the [Elastic Package Registry (EPM)](https://github.com/elastic/package-registry).

In the future it may also contain code for validating said specifications.

# Purpose

Please use this repository to discuss any changes to the specification, either my making issues or PRs to the specification.

# Specification Format

An Elastic Package specification is determined by the package's folder structure, folder names, the presence of certain files within these folders, and the structure of those files' contents.

Each specification format is versioned. You will find folders at the root of this repository for each active version of the specification format. 

Within each version folder, there must be a `spec.yml` file. This file is the entry point for the specification for a package's contents. For readability, the specification file may (recursively) reference other specification files via `specRef` attributes.

