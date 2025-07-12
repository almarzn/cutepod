package chart

import (
	"fmt"

	"github.com/goccy/go-yaml"
)

// Lint parses and renders chart templates and validates resulting YAML.
func Lint(opts LintOptions) {
	_, err := Parse(opts)
	if err != nil {
		fmt.Println(yaml.FormatError(err, true, true))
		return
	}

	fmt.Println("Chart is valid.")
}
