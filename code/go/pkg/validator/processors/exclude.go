package processors

import (
	"regexp"

	"github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

type Exclude struct {
	pattern *regexp.Regexp
}

func NewExclude(pattern string) *Exclude {
	var patternRe *regexp.Regexp
	if pattern != "" {
		patternRe = regexp.MustCompile(pattern)
	}

	return &Exclude{
		pattern: patternRe,
	}
}

func (p Exclude) Name() string {
	return "exclude"
}

func (p Exclude) Process(issues errors.ValidationErrors) (errors.ValidationErrors, error) {
	if p.pattern == nil {
		return issues, nil
	}

	return issues.Filter(func(i *errors.ValidationError) bool {
		return !p.pattern.MatchString((*i).Error())
	}), nil
}
