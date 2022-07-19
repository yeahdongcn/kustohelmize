package config

type FieldStrategy string

const (
	FieldStrategyString FieldStrategy = "string"
	FieldStrategyIf     FieldStrategy = "if"
	FieldStrategyXXX    FieldStrategy = "xxx"
)

type File struct {
	XPaths map[string]FieldStrategy
}

type Config struct {
	Release string `yaml:"release"`
	Files   []File `yaml:"files"`
}
