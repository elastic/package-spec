import "github.com/elastic/package-spec/spec:spec"

spec_version_major: int

#Manifest: {
  format_version: spec.Version
  name: spec.PackageName
  type: "integration"
  title: string
  description: string
  version: spec.Version
  source?: spec.Source
  release?: spec.Release
  categories?: [...spec.Category]
  conditions?: spec.Conditions
  policy_templates?: [...PolicyTemplate]
  icons?: [...spec.Icon]
  vars?: [...spec.Variable]
  screenshots?: [...spec.Screenshot]
  owner: spec.Owner
  elasticsearch?: privileges?: cluster?: [...string]

  if spec_version_major < 2 {
    // Deprecated
    license?: spec.Subscription
  }
}

PolicyTemplate: {
  name: string
  title: string
  description: string
  categories?: [...spec.Category]
  data_streams?: [...string]
  inputs?: [...Input]
  multiple?: bool
  icons?: [...spec.Icon]
  screenshots?: [...spec.Screenshot]
  vars?: [...spec.Variable]
}

Input: {
  type: string
  title: string
  description: string
  template_path?: string
  input_group?: string
  multi?: *false | bool
  vars?: [...spec.Variable]
}
