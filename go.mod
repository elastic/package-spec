module github.com/elastic/package-spec

go 1.17

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/creasty/defaults v1.5.2
	github.com/elastic/go-licenser v0.3.1
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
)

require (
	github.com/PaesslerAG/gval v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	golang.org/x/tools v0.0.0-20200130002326-2f3ba24bd6e7 // indirect
)

replace github.com/xeipuuv/gojsonschema => github.com/jsoriano/gojsonschema v1.2.1-0.20220622174425-9a973da8ffff
