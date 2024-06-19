package util

import (
	"strconv"
	"strings"
)

type String string

func (s String) IsBool() bool {
	return s == "true" || s == "false"
}

func (s String) IsNumeric() bool {
	_, err := strconv.ParseFloat(string(s), 64)
	return err == nil
}

func (s String) HasNewLine() bool {
	return strings.Contains(string(s), "\n")
}

func (s String) IsWhiteSpace() bool {
	return len(strings.TrimSpace(string(s))) == 0
}
