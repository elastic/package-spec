package spec

Category: "aws" | "azure" | "cloud" | "config_management" | "containers" | "crm" | "custom" | "datastore" | "elastic_stack" | "google_cloud" | "kubernetes" | "languages" | "message_queue" | "monitoring" | "network" | "notification" | "os_system" | "productivity" | "security" | "support" | "threat_intel" | "ticketing" | "version_control" | "web"

Conditions: {
  elastic?: {
    subscription?: Subscription
  }
  kibana?: {
    version?: Version
  }
}

DataType: "bool" | "email" | "integer" | "password" | "text" | "textarea" | "time_zone" | "url" | "yaml"

Icon: {
  src: RelativePath
  title?: string
  size?: string
  type?: string
  dark_mode?: bool
}

InputVariableValue: null | string | int | bool | [...InputVariableValue]

License: "Apache-2.0" | "Elastic-2.0"

PackageName: =~"^[a-z0-9_]+$"

RelativePath: string // TODO: How to implement this?

Release: "ga" | "beta" | "experimental"

Owner: github: =~"^(([a-zA-Z0-9-]+)|([a-zA-Z0-9-]+\/[a-zA-Z0-9-]+))$"

Screenshot: {
  src: RelativePath
  title: string
  size?: string
  type?: string
}

Source: {
  license?: License
}

Subscription: "basic" | "gold" | "platinum" | "enterprise"

Variable:
  name: string
  type: DataType
  title?: string
  description?: string
  multi?: *false | bool
  required?: *false | bool
  show_user?: *false | bool
  url_allowed_schemes?: [...string]
  default: InputVariableValue

Version: =~"^([0-9]+).([0-9]+).([0-9]+)(?:-([0-9A-Za-z-]+(?:.[0-9A-Za-z-]+)*))?(?:+[0-9A-Za-z-]+)?$"
