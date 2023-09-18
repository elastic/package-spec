package processors

import "github.com/elastic/package-spec/v2/code/go/pkg/errors"

type Processor interface {
	Process(issues errors.ValidationErrors) (errors.ValidationErrors, error)
	Name() string
}
