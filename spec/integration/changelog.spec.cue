import "github.com/elastic/package-spec/spec:spec"

#Changelog: [...ChangelogEntry]

ChangelogEntry: {
  version: spec.Version
  changes: [...{
    description: string
    type: "breaking-change" | "bugfix" | "enhancement"
    link: string
  }]
}
