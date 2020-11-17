package validator

func adjustErrorDescription(description string) string {
	if description == "Does not match format 'relative-path'" {
		return "relative path is invalid or target doesn't exist"
	}
	return description
}