// SPDX-License-Identifier: GPL-3.0-or-later
package scaffold

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// orderedObject is a JSON object that remembers the order its top-level keys
// were read in, and preserves that order (existing keys first, new ones
// appended) when marshaled back. Repairing metadata.json round-trips
// through this instead of a plain map[string]json.RawMessage so the rewrite
// doesn't alphabetize every key and turn a two-key fix into a full-file diff.
type orderedObject struct {
	fields []orderedField
	index  map[string]int
}

type orderedField struct {
	Key   string
	Value json.RawMessage
}

// newOrderedObject parses a JSON object, preserving key order and the exact
// bytes of every value.
func newOrderedObject(content []byte) (*orderedObject, error) {
	dec := json.NewDecoder(bytes.NewReader(content))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected a JSON object")
	}

	obj := &orderedObject{index: map[string]int{}}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected non-string object key %v", keyTok)
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		obj.index[key] = len(obj.fields)
		obj.fields = append(obj.fields, orderedField{Key: key, Value: value})
	}
	if _, err := dec.Token(); err != nil { // consume the closing '}'
		return nil, err
	}
	return obj, nil
}

func (o *orderedObject) Get(key string) (json.RawMessage, bool) {
	i, ok := o.index[key]
	if !ok {
		return nil, false
	}
	return o.fields[i].Value, true
}

// Set overwrites key's value in place if it's already present, or appends it
// as a new field otherwise.
func (o *orderedObject) Set(key string, value json.RawMessage) {
	if i, ok := o.index[key]; ok {
		o.fields[i].Value = value
		return
	}
	o.index[key] = len(o.fields)
	o.fields = append(o.fields, orderedField{Key: key, Value: value})
}

// MarshalJSON implements json.Marshaler. encoding/json uses a Marshaler's
// output verbatim (aside from whitespace), so the field order built here
// survives json.MarshalIndent.
func (o *orderedObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, f := range o.fields {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, err := marshalNoEscape(f.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')
		buf.Write(f.Value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// marshalNoEscape is json.Marshal without HTML-escaping &, <, and >, matching
// the encoder settings module.Metadata.Write uses.
func marshalNoEscape(v any) (json.RawMessage, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return json.RawMessage(bytes.TrimRight(buf.Bytes(), "\n")), nil
}
