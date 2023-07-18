module github.com/elastic/package-spec/compliance

go 1.20

require (
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/cucumber/godog v0.12.7-0.20230607093746-72db47c51993
	github.com/cucumber/messages/go/v21 v21.0.1
	github.com/elastic/package-spec/v2 v2.9.0
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/cucumber/gherkin/go/v26 v26.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.4 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/elastic/package-spec/v2 => ../
