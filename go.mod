module github.com/newrelic/nri-jmx

go 1.16

require (
	github.com/coreos/bbolt v1.3.2 // indirect
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/golangci/golangci-lint v1.43.0
	github.com/iancoleman/strcase v0.2.0
	github.com/kr/pretty v0.2.1
	github.com/newrelic/infra-integrations-sdk v3.7.0+incompatible
	github.com/newrelic/nrjmx/gojmx v0.0.0-20220104151522-5634d55e8419
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/sanposhiho/wastedassign v0.2.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tomarrell/wrapcheck v1.0.0 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/newrelic/nrjmx/gojmx => /Users/cciutea/workspace/nr/int/nrjmx/gojmx
