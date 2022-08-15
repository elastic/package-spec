import "github.com/elastic/package-spec/spec:spec"

#Manifest: {
  format_version: spec.Version
  name: spec.PackageName
  type: "integration"
  title: string
  description: string
  version: spec.Version
  source?: spec.Source
  license?: spec.Subscription
  release?: spec.Release
  categories?: [...spec.Category]
  conditions?: spec.Conditions
  policy_templates?: [...PolicyTemplate]
  icons?: [...spec.Icon]
  screenshots?: [...spec.Screenshot]
  owner: spec.Owner
  elasticsearch?: privileges?: cluster?: [...string]
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
  vars: [...spec.Variable]
}

Input: {
  type: string
  title: string
  description: string
  template_path?: string
  input_group?: string
  multi?: *false | bool
}
