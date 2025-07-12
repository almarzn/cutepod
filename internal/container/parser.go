package container

import (
	"bytes"
	"cutepod/internal/meta"
	"fmt"

	"github.com/goccy/go-yaml"
)

func CanParse(spec *meta.BaseSpec) bool {
	return spec.Kind == "CuteContainer"
}

func Parse(buf bytes.Buffer) (*CuteContainer, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(buf.Bytes()))

	var doc CuteContainer
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s", yaml.FormatError(err, true, true))
	}

	errors := validateWithAnnotation(buf.String(), doc)
	if len(errors) != 0 {
		for _, e := range errors {
			fmt.Println(e)
		}

		return nil, fmt.Errorf("there were errors while parsing the container")
	}

	return &doc, nil
}
