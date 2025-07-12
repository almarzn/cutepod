package meta

import (
	"bytes"
	"fmt"

	"github.com/goccy/go-yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BaseSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func Parse(buf bytes.Buffer) (*BaseSpec, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(buf.Bytes()))

	var doc BaseSpec
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s", yaml.FormatError(err, true, true))
	}

	return &doc, nil
}
