package semantic

// ValidateKibanaNoDanglingObjectIDs returns validation errors if there are any
// dangling references to Kibana objects in any Kibana object files. That is, it
// returns validation errors if a Kibana object file in the package references another
// Kibana object with ID i, but no Kibana object file for object ID i is found in the
// package.
func ValidateKibanaNoDanglingObjectIDs(pkgRoot string) error {
	// TODO: will be implemented in follow up PR
	return nil
}
