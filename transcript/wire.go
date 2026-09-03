package transcript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/openproficiency/opm-go/internal/canonical"
	"github.com/openproficiency/opm-go/internal/schema"
	"gopkg.in/yaml.v3"
)

type wireEntry struct {
	Schema           string    `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	UserEmail        string    `json:"user-email" yaml:"user-email"`
	Topic            string    `json:"topic" yaml:"topic"`
	TopicList        string    `json:"topic-list" yaml:"topic-list"`
	TopicListVersion string    `json:"topic-list-version" yaml:"topic-list-version"`
	TopicListOwner   string    `json:"topic-list-owner" yaml:"topic-list-owner"`
	TopicListSources *[]string `json:"topic-list-sources,omitempty" yaml:"topic-list-sources,omitempty"`
	Score            string    `json:"score" yaml:"score"`
	IssuedAt         string    `json:"issued-at" yaml:"issued-at"`
	ValidUntil       string    `json:"valid-until" yaml:"valid-until"`
	IssuedBy         string    `json:"issued-by" yaml:"issued-by"`
	VerificationURL  string    `json:"verification-url,omitempty" yaml:"verification-url,omitempty"`
	Signature        *string   `json:"signature" yaml:"signature"`
	SignedBy         *string   `json:"signed-by" yaml:"signed-by"`
}

// MarshalJSON returns an OPM transcript entry encoded as JSON.
func (entry Entry) MarshalJSON() ([]byte, error) {
	wire, signed, err := entry.toWire()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("marshal transcript entry JSON: %w", err)
	}
	if signed {
		if err := schema.ValidateJSON(schema.TranscriptEntry, data); err != nil {
			return nil, fmt.Errorf("marshal transcript entry JSON: %w", err)
		}
	}

	return data, nil
}

// UnmarshalJSON loads an OPM transcript entry encoded as JSON.
func (entry *Entry) UnmarshalJSON(data []byte) error {
	if entry == nil {
		return fmt.Errorf("unmarshal transcript entry JSON: entry is nil")
	}

	var wire wireEntry
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return fmt.Errorf("unmarshal transcript entry JSON: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return fmt.Errorf("unmarshal transcript entry JSON: %w", err)
	}

	decoded, err := entryFromWire(wire)
	if err != nil {
		return fmt.Errorf("unmarshal transcript entry JSON: %w", err)
	}
	if decoded.signature != "" {
		if err := schema.ValidateJSON(schema.TranscriptEntry, data); err != nil {
			return fmt.Errorf("unmarshal transcript entry JSON: %w", err)
		}
	}

	*entry = decoded
	return nil
}

// MarshalYAML returns an OPM transcript entry encoded as YAML.
func (entry Entry) MarshalYAML() ([]byte, error) {
	wire, signed, err := entry.toWire()
	if err != nil {
		return nil, err
	}

	data, err := encodeYAML(wire)
	if err != nil {
		return nil, fmt.Errorf("marshal transcript entry YAML: %w", err)
	}
	if signed {
		if err := schema.ValidateYAML(schema.TranscriptEntry, data); err != nil {
			return nil, fmt.Errorf("marshal transcript entry YAML: %w", err)
		}
	}

	return data, nil
}

// UnmarshalYAML loads an OPM transcript entry encoded as YAML.
func (entry *Entry) UnmarshalYAML(data []byte) error {
	if entry == nil {
		return fmt.Errorf("unmarshal transcript entry YAML: entry is nil")
	}

	jsonData, err := yamlToJSON(data)
	if err != nil {
		return fmt.Errorf("unmarshal transcript entry YAML: %w", err)
	}
	if err := entry.UnmarshalJSON(jsonData); err != nil {
		return fmt.Errorf("unmarshal transcript entry YAML: %w", err)
	}

	return nil
}

func (entry Entry) toWire() (wireEntry, bool, error) {
	if err := entry.validate(); err != nil {
		return wireEntry{}, false, err
	}

	signature, signedBy, signed, err := entry.currentSignature()
	if err != nil {
		return wireEntry{}, false, err
	}
	if signed && !emailDomainMatches(signedBy, entry.IssuedBy) {
		return wireEntry{}, false, fmt.Errorf("%w: signed by %q, issued by %q", ErrIssuerMismatch, signedBy, entry.IssuedBy)
	}

	var sources *[]string
	if entry.TopicListSources != nil {
		copiedSources := make([]string, len(entry.TopicListSources))
		copy(copiedSources, entry.TopicListSources)
		sources = &copiedSources
	}

	wire := wireEntry{
		Schema:           entry.schemaURL,
		UserEmail:        entry.UserEmail,
		Topic:            entry.Topic,
		TopicList:        entry.TopicList,
		TopicListVersion: entry.TopicListVersion,
		TopicListOwner:   entry.TopicListOwner,
		TopicListSources: sources,
		Score:            entry.Score.String(),
		IssuedAt:         formatTime(entry.IssuedAt),
		ValidUntil:       formatTime(entry.ValidUntil),
		IssuedBy:         entry.IssuedBy,
		VerificationURL:  entry.VerificationURL,
	}
	if signed {
		wire.Signature = &signature
		wire.SignedBy = &signedBy
		data, err := json.Marshal(wire)
		if err != nil {
			return wireEntry{}, false, fmt.Errorf("validate signed transcript entry: %w", err)
		}
		if err := schema.ValidateJSON(schema.TranscriptEntry, data); err != nil {
			return wireEntry{}, false, fmt.Errorf("validate signed transcript entry: %w", err)
		}
	}

	return wire, signed, nil
}

func entryFromWire(wire wireEntry) (Entry, error) {
	parsedScore, err := parseScore(wire.Score)
	if err != nil {
		return Entry{}, fmt.Errorf("decode score: %w", err)
	}
	issuedAt, err := time.Parse(time.RFC3339, wire.IssuedAt)
	if err != nil {
		return Entry{}, fmt.Errorf("decode issued-at timestamp: %w", err)
	}
	validUntil, err := time.Parse(time.RFC3339, wire.ValidUntil)
	if err != nil {
		return Entry{}, fmt.Errorf("decode valid-until timestamp: %w", err)
	}
	if (wire.Signature == nil) != (wire.SignedBy == nil) {
		return Entry{}, fmt.Errorf("signature and signed-by must both be strings or null")
	}

	entry := Entry{
		UserEmail:        wire.UserEmail,
		Topic:            wire.Topic,
		TopicList:        wire.TopicList,
		TopicListVersion: wire.TopicListVersion,
		TopicListOwner:   wire.TopicListOwner,
		Score:            parsedScore,
		IssuedAt:         issuedAt,
		ValidUntil:       validUntil,
		IssuedBy:         wire.IssuedBy,
		VerificationURL:  wire.VerificationURL,
		schemaURL:        wire.Schema,
	}
	if wire.TopicListSources != nil {
		entry.TopicListSources = make([]string, len(*wire.TopicListSources))
		copy(entry.TopicListSources, *wire.TopicListSources)
	}
	if wire.Signature != nil {
		if *wire.Signature == "" || *wire.SignedBy == "" {
			return Entry{}, fmt.Errorf("signature and signed-by must not be empty")
		}
		entry.signature = *wire.Signature
		entry.signedBy = *wire.SignedBy
	}
	if err := entry.validate(); err != nil {
		return Entry{}, err
	}
	if entry.signature != "" {
		protectedState, err := canonical.NewState(entry.protected())
		if err != nil {
			return Entry{}, fmt.Errorf("capture transcript entry protected state: %w", err)
		}
		entry.protectedState = protectedState
	}

	return entry, nil
}

func (entry Entry) currentSignature() (string, string, bool, error) {
	if entry.signature == "" || entry.signedBy == "" {
		return "", "", false, nil
	}

	matches, err := entry.protectedState.Matches(entry.protected())
	if err != nil {
		return "", "", false, fmt.Errorf("compare transcript entry protected state: %w", err)
	}
	if !matches {
		return "", "", false, nil
	}

	return entry.signature, entry.signedBy, true, nil
}

func encodeYAML(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

func yamlToJSON(data []byte) ([]byte, error) {
	var document any
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	if err := ensureYAMLEOF(decoder); err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values are not allowed")
		}
		return err
	}

	return nil
}

func ensureYAMLEOF(decoder *yaml.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple YAML documents are not allowed")
		}
		return err
	}

	return nil
}

func emailDomainMatches(email, issuer string) bool {
	at := strings.LastIndexByte(email, '@')
	return at >= 0 && email[at+1:] == issuer
}
