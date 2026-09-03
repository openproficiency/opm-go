package transcript

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/openproficiency/opm-go/internal/schema"
)

// Entries is a portable collection of transcript entries.
type Entries []Entry

// MarshalJSON returns an OPM transcript encoded as JSON.
func (entries Entries) MarshalJSON() ([]byte, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("marshal transcript JSON: at least one entry is required")
	}

	wires := make([]wireEntry, len(entries))
	allSigned := true
	for index, entry := range entries {
		wire, signed, err := entry.toWire()
		if err != nil {
			return nil, fmt.Errorf("marshal transcript JSON entry %d: %w", index, err)
		}
		wires[index] = wire
		allSigned = allSigned && signed
	}

	data, err := json.Marshal(wires)
	if err != nil {
		return nil, fmt.Errorf("marshal transcript JSON: %w", err)
	}
	if allSigned {
		if err := schema.ValidateJSON(schema.Transcript, data); err != nil {
			return nil, fmt.Errorf("marshal transcript JSON: %w", err)
		}
	}

	return data, nil
}

// UnmarshalJSON loads an OPM transcript encoded as JSON.
func (entries *Entries) UnmarshalJSON(data []byte) error {
	if entries == nil {
		return fmt.Errorf("unmarshal transcript JSON: entries is nil")
	}

	var rawEntries []json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&rawEntries); err != nil {
		return fmt.Errorf("unmarshal transcript JSON: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return fmt.Errorf("unmarshal transcript JSON: %w", err)
	}
	if len(rawEntries) == 0 {
		return fmt.Errorf("unmarshal transcript JSON: at least one entry is required")
	}

	decoded := make(Entries, len(rawEntries))
	for index, rawEntry := range rawEntries {
		if err := decoded[index].UnmarshalJSON(rawEntry); err != nil {
			return fmt.Errorf("unmarshal transcript JSON entry %d: %w", index, err)
		}
	}

	*entries = decoded
	return nil
}

// MarshalYAML returns an OPM transcript encoded as YAML.
func (entries Entries) MarshalYAML() ([]byte, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("marshal transcript YAML: at least one entry is required")
	}

	wires := make([]wireEntry, len(entries))
	allSigned := true
	for index, entry := range entries {
		wire, signed, err := entry.toWire()
		if err != nil {
			return nil, fmt.Errorf("marshal transcript YAML entry %d: %w", index, err)
		}
		wires[index] = wire
		allSigned = allSigned && signed
	}

	data, err := encodeYAML(wires)
	if err != nil {
		return nil, fmt.Errorf("marshal transcript YAML: %w", err)
	}
	if allSigned {
		if err := schema.ValidateYAML(schema.Transcript, data); err != nil {
			return nil, fmt.Errorf("marshal transcript YAML: %w", err)
		}
	}

	return data, nil
}

// UnmarshalYAML loads an OPM transcript encoded as YAML.
func (entries *Entries) UnmarshalYAML(data []byte) error {
	if entries == nil {
		return fmt.Errorf("unmarshal transcript YAML: entries is nil")
	}

	jsonData, err := yamlToJSON(data)
	if err != nil {
		return fmt.Errorf("unmarshal transcript YAML: %w", err)
	}
	if err := entries.UnmarshalJSON(jsonData); err != nil {
		return fmt.Errorf("unmarshal transcript YAML: %w", err)
	}

	return nil
}
