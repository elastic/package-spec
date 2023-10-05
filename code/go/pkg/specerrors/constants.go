// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specerrors

// Constants to be used for the structured errors
const (
	UnassignedCode = ""

	// PSR - Package Spec [General] Rule
	CodeNonGASpecOnGAPackage         = "PSR00001"
	CodePrereleaseFeatureOnGAPackage = "PSR00002"

	// SVR - Semantic Validation Rules
	CodeKibanaDashboardWithQueryButNoFilter = "SVR00001"
	CodeKibanaDashboardWithoutFilter        = "SVR00002"
	CodeKibanaDanglingObjectsIDs            = "SVR00003"
	CodeVisualizationByValue                = "SVR00004"
	CodeMinimumKibanaVersion                = "SVR00005"
)
