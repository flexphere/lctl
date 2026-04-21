package yamlplist

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Encode renders the Document as YAML. 2-space indent.
func Encode(d *Document) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(d); err != nil {
		return nil, fmt.Errorf("encode yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("close yaml: %w", err)
	}
	return buf.Bytes(), nil
}

// Decode parses YAML into a Document. Unknown keys cause an error so
// the user notices typos.
func Decode(data []byte) (*Document, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	d := &Document{}
	if err := dec.Decode(d); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}
	return d, nil
}
