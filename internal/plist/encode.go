package plist

import (
	"bytes"
	"fmt"
	"io"

	hplist "howett.net/plist"
)

// Encode writes the agent as an XML plist to w.
func Encode(w io.Writer, a *Agent) error {
	enc := hplist.NewEncoderForFormat(w, hplist.XMLFormat)
	enc.Indent("\t")
	if err := enc.Encode(a); err != nil {
		return fmt.Errorf("encode plist: %w", err)
	}
	return nil
}

// EncodeBytes encodes the agent to a byte slice.
func EncodeBytes(a *Agent) ([]byte, error) {
	var buf bytes.Buffer
	if err := Encode(&buf, a); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode parses an XML plist from r into a new Agent.
func Decode(r io.Reader) (*Agent, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read plist: %w", err)
	}
	return DecodeBytes(data)
}

// DecodeBytes parses an XML plist from data into a new Agent.
func DecodeBytes(data []byte) (*Agent, error) {
	a := &Agent{}
	dec := hplist.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(a); err != nil {
		return nil, fmt.Errorf("decode plist: %w", err)
	}
	return a, nil
}
