package config

type FieldStrategy string

const (
	FieldStrategyPlain   FieldStrategy = "plain-value"
	FieldStrategyNewline FieldStrategy = "newline-value"
	FieldStrategyIf      FieldStrategy = "expansion-if-present"
	FieldStrategyXXX     FieldStrategy = "xxx"
)

type XXX struct {
	FieldStrategy FieldStrategy
	Value         string
}

type XPaths map[string]XXX

type Config struct {
	ChartName string            `yaml:"chartName"`
	Rules     map[string]XPaths `yaml:"rules"`
}
