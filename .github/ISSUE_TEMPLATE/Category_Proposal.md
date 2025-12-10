---
name: Category Proposal
about: For proposing and discussing a change to the list of categories available under the Elastic Package Specification
labels: [discuss, Team:Ecosystem]
title: "[Category Proposal]"
---

*Please read the section on [Category Proposals in the Contributing Guide](../../CONTRIBUTING.md#category-proposals) and flesh out this issue accordingly. Thank you!*

## Context

<!-- Provide some background context on why this new category is needed. -->

## Eligibility Requirements

Please, address the following to make sure the propsal is elegible:

- [ ] New category must group at least 5 integrations, existing or planned. List them here.
- [ ] Alternative existing categories in which these integrations are (currently) or could be (in the future) grouped.
- [ ] Proof that this label is a well-known industry term and not jargon used by a few.
  - Usage of the keyword in Gartner or Forrester industry reports.
  - Industry blogs using this label or keyword.
  - Customer mentioning this keyword.
- [ ] Proof that users use this label to describe this subset of integrations.
  - Enhancement requests that complain about the lack of this category.
  - Full-story data reporting on average 50+ unique searches for this keyword per month in the integrations UI.
  - Usage of this category for filtering purposes by our industry.
- [ ] Impact that are we expecting with this category and how we will measure it (success criteria)

## Implementation checklist

- [ ] Support in package-spec manifest categories
- [ ] Support in package-registry /categories API
- [ ] Support in Fleet
- [ ] Update integrations dependencies to support new category
