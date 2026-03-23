// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var errGithubIssue = errors.New("issue number in changelog link should be a positive number") // TODO test validationError structuredError

// ChangelogLinkError records the link and the error
type ChangelogLinkError struct {
	link string
	err  error
}

// Is checks if the target matches one of the allowed links errors.
// Currently checks for github issue/pr links.
func (e ChangelogLinkError) Is(target error) bool {
	return target == errGithubIssue
}

func (e ChangelogLinkError) Error() string {
	return fmt.Sprintf("%s: %s", e.link, e.err.Error())
}

// ValidateChangelogLinks returns validation errors if the link(s) do not have a valid PR github.com link.
// If the link is not a github.com link this validation is skipped and does not return an error.
func ValidateChangelogLinks(fsys fspath.FS) specerrors.ValidationErrors {
	changelogLinks, err := readChangelogLinks(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}
	return ensureLinksAreValid(changelogLinks)
}

func readChangelogLinks(fsys fspath.FS) ([]string, error) {
	return readChangelog(fsys, `$[*].changes[*].link`)
}

func ensureLinksAreValid(links []string) specerrors.ValidationErrors {
	type validateFn func(link *url.URL) error
	var errs specerrors.ValidationErrors

	validateLinks := []struct {
		domain       string
		validateLink validateFn
	}{
		{
			"github.com",
			validateGithubLink,
		},
	}
	for _, link := range links {
		linkURL, err := url.Parse(link)
		if err != nil {
			errs.Append(specerrors.ValidationErrors{
				specerrors.NewStructuredErrorf("invalid URL %v", err),
			})
			continue
		}
		for _, vl := range validateLinks {
			if strings.Contains(linkURL.Host, vl.domain) {
				if err = vl.validateLink(linkURL); err != nil {
					errs.Append(specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)})
				}
			}
		}
	}
	return errs
}

func validateGithubLink(ghLink *url.URL) error {
	prNum, err := strconv.Atoi(path.Base(ghLink.Path))
	if err != nil || prNum <= 0 {
		return &ChangelogLinkError{
			ghLink.String(),
			errGithubIssue,
		}
	}
	return nil
}
