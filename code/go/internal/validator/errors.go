package validator

import (
	"fmt"
	"strings"
)

// ValidationErrors is an Error that contains a iterable collection of validation error messages.
type ValidationErrors []string

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "found 0 validation errors"
	}

	var message strings.Builder
	errorWord := "errors"
	if len(ve) == 1 {
		errorWord = "error"
	}
	fmt.Fprintf(&message, "found %v validation %v:\n", len(ve), errorWord)
	for _, err := range ve {
		fmt.Fprintf(&message, "\t%v\n", err)
	}

	return message.String()
}
