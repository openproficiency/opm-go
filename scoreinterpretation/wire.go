package scoreinterpretation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/openproficiency/opm-go/internal/canonical"
	"github.com/openproficiency/opm-go/internal/schema"
	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/topic"
	"gopkg.in/yaml.v3"
)

type wireList struct {
	Schema          string                        `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Owner           string                        `json:"owner" yaml:"owner"`
	Name            string                        `json:"name" yaml:"name"`
	Description     string                        `json:"description" yaml:"description"`
	Version         string                        `json:"version" yaml:"version"`
	IssuedAt        string                        `json:"issued-at" yaml:"issued-at"`
	Signature       *string                       `json:"signature" yaml:"signature"`
	SignedBy        *string                       `json:"signed-by" yaml:"signed-by"`
	Interpretations map[string]wireInterpretation `json:"score-interpretations" yaml:"score-interpretations"`
	Dependencies    map[string]any                `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

type wireInterpretation struct {
	Schema       string         `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	ID           string         `json:"id,omitempty" yaml:"id,omitempty"`
	Name         string         `json:"name" yaml:"name"`
	Description  string         `json:"description" yaml:"description"`
	Requirements map[string]any `json:"requirements" yaml:"requirements"`
}

type wireDependency struct {
	Owner     string    `json:"topic-list-owner" yaml:"topic-list-owner"`
	Name      string    `json:"topic-list-name" yaml:"topic-list-name"`
	Version   string    `json:"topic-list-version" yaml:"topic-list-version"`
	Locations *[]string `json:"locations,omitempty" yaml:"locations,omitempty"`
}

type protectedList struct {
	Owner           string                             `json:"owner"`
	Name            string                             `json:"name"`
	Description     string                             `json:"description"`
	Version         string                             `json:"version"`
	IssuedAt        string                             `json:"issued-at"`
	Interpretations map[string]protectedInterpretation `json:"score-interpretations"`
	Dependencies    map[string]protectedDependency     `json:"dependencies,omitempty"`
}

type protectedInterpretation struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Requirements map[string]any `json:"requirements"`
}

type protectedDependency struct {
	Owner   string `json:"topic-list-owner"`
	Name    string `json:"topic-list-name"`
	Version string `json:"topic-list-version"`
}

type rawList struct {
	Schema          string                     `json:"$schema"`
	Owner           string                     `json:"owner"`
	Name            string                     `json:"name"`
	Description     string                     `json:"description"`
	Version         string                     `json:"version"`
	IssuedAt        string                     `json:"issued-at"`
	Signature       *string                    `json:"signature"`
	SignedBy        *string                    `json:"signed-by"`
	Interpretations map[string]json.RawMessage `json:"score-interpretations"`
	Dependencies    map[string]json.RawMessage `json:"dependencies"`
}

type rawInterpretation struct {
	Schema       string          `json:"$schema"`
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Requirements json.RawMessage `json:"requirements"`
}

type rawDependency struct {
	Owner     string    `json:"topic-list-owner"`
	Name      string    `json:"topic-list-name"`
	Version   string    `json:"topic-list-version"`
	Locations *[]string `json:"locations"`
}

// MarshalJSON returns deterministic OPM-compatible JSON.
func (list List) MarshalJSON() ([]byte, error) {
	if err := list.Validate(); err != nil {
		return nil, err
	}
	document, err := list.wireDocument(true)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("marshal score interpretation list JSON: %w", err)
	}
	return data, nil
}

// UnmarshalJSON loads an OPM-compatible JSON score interpretation list.
func (list *List) UnmarshalJSON(data []byte) error {
	if list == nil {
		return errors.New("unmarshal score interpretation list JSON: nil receiver")
	}
	if err := validateListSchemaExceptDependencies(data); err != nil {
		return err
	}

	decoded, err := decodeJSON(data)
	if err != nil {
		return err
	}

	if err := decoded.Validate(); err != nil {
		return err
	}
	if decoded.signature != nil {
		protected, protectedErr := decoded.protectedDocument()
		if protectedErr != nil {
			return fmt.Errorf("capture score interpretation list signature state: %w", protectedErr)
		}
		state, stateErr := canonical.NewState(protected)
		if stateErr != nil {
			return fmt.Errorf("capture score interpretation list signature state: %w", stateErr)
		}
		decoded.signatureState = state
	}

	*list = decoded
	return nil
}

func validateListSchemaExceptDependencies(data []byte) error {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("decode score interpretation list for schema validation: %w", err)
	}
	delete(document, "dependencies")

	normalized, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("normalize score interpretation list for schema validation: %w", err)
	}
	if err := schema.ValidateJSON(schema.ScoreInterpretationList, normalized); err != nil {
		return err
	}
	return nil
}

// MarshalYAML returns deterministic OPM-compatible YAML.
func (list List) MarshalYAML() ([]byte, error) {
	if err := list.Validate(); err != nil {
		return nil, err
	}
	document, err := list.wireDocument(true)
	if err != nil {
		return nil, err
	}
	data, err := encodeYAML(document)
	if err != nil {
		return nil, fmt.Errorf("marshal score interpretation list YAML: %w", err)
	}
	return data, nil
}

// UnmarshalYAML loads an OPM-compatible YAML score interpretation list.
func (list *List) UnmarshalYAML(data []byte) error {
	if list == nil {
		return errors.New("unmarshal score interpretation list YAML: nil receiver")
	}
	jsonData, err := yamlToJSON(data)
	if err != nil {
		return fmt.Errorf("unmarshal score interpretation list YAML: %w", err)
	}
	if err := list.UnmarshalJSON(jsonData); err != nil {
		return fmt.Errorf("unmarshal score interpretation list YAML: %w", err)
	}
	return nil
}

func (list List) wireDocument(includeSignature bool) (wireList, error) {
	interpretations := make(map[string]wireInterpretation, len(list.Interpretations))
	for id, interpretation := range list.Interpretations {
		wire, err := makeWireInterpretation(interpretation, interpretation.ID == id)
		if err != nil {
			return wireList{}, err
		}
		interpretations[id] = wire
	}

	dependencies := make(map[string]any, len(list.Dependencies))
	for alias, dependency := range list.Dependencies {
		dependencies[alias] = dependencyWireValue(
			dependency,
			list.dependencyLongForm[alias],
			list.dependencyLocationsPresent[alias],
		)
	}
	if len(dependencies) == 0 {
		dependencies = nil
	}

	var signature *string
	var signedBy *string
	if includeSignature && list.checkSignature() == nil {
		signature = list.signature
		signedBy = list.signedBy
	}

	return wireList{
		Schema:          list.schemaURL,
		Owner:           list.Owner,
		Name:            list.Name,
		Description:     list.Description,
		Version:         list.Version,
		IssuedAt:        formatTime(list.IssuedAt),
		Signature:       signature,
		SignedBy:        signedBy,
		Interpretations: interpretations,
		Dependencies:    dependencies,
	}, nil
}

func makeWireInterpretation(
	interpretation Interpretation,
	omitMatchingID bool,
) (wireInterpretation, error) {
	requirements, err := encodeRequirements(interpretation.Requirements)
	if err != nil {
		return wireInterpretation{}, err
	}

	id := interpretation.ID
	if omitMatchingID {
		id = ""
	}
	return wireInterpretation{
		Schema:       interpretation.schemaURL,
		ID:           id,
		Name:         interpretation.Name,
		Description:  interpretation.Description,
		Requirements: requirements,
	}, nil
}

func encodeRequirements(requirements []Requirement) (map[string]any, error) {
	encoded := make(map[string]any, len(requirements))
	for _, requirement := range requirements {
		key, value, err := encodeRequirement(requirement)
		if err != nil {
			return nil, err
		}
		if _, exists := encoded[key]; exists {
			return nil, fmt.Errorf("duplicate requirement key %q", key)
		}
		encoded[key] = value
	}
	return encoded, nil
}

func encodeRequirement(requirement Requirement) (string, any, error) {
	switch typed := requirement.(type) {
	case requiredTopic:
		return typed.key(), typed.minimum.String(), nil
	case All:
		return encodeAll(typed.ID, typed.Requirements)
	case *All:
		if typed == nil {
			return "", nil, errors.New("all requirement is nil")
		}
		return encodeAll(typed.ID, typed.Requirements)
	case Any:
		return encodeAny(typed.ID, typed.Requirements)
	case *Any:
		if typed == nil {
			return "", nil, errors.New("any requirement is nil")
		}
		return encodeAny(typed.ID, typed.Requirements)
	case AtLeast:
		return encodeAtLeast(typed.ID, typed.MinCount, typed.Requirements)
	case *AtLeast:
		if typed == nil {
			return "", nil, errors.New("at-least requirement is nil")
		}
		return encodeAtLeast(typed.ID, typed.MinCount, typed.Requirements)
	default:
		return "", nil, fmt.Errorf("unsupported requirement type %T", requirement)
	}
}

func encodeAll(id string, requirements []Requirement) (string, any, error) {
	key, err := allKey(id)
	if err != nil {
		return "", nil, err
	}
	encoded, err := encodeRequirements(requirements)
	return key, encoded, err
}

func encodeAny(id string, requirements []Requirement) (string, any, error) {
	key, err := anyKey(id)
	if err != nil {
		return "", nil, err
	}
	encoded, err := encodeRequirements(requirements)
	return key, encoded, err
}

func encodeAtLeast(id string, minCount int, requirements []Requirement) (string, any, error) {
	key, err := atLeastKey(id, minCount)
	if err != nil {
		return "", nil, err
	}
	encoded, err := encodeRequirements(requirements)
	return key, encoded, err
}

func (list List) protectedDocument() (protectedList, error) {
	interpretations := make(map[string]protectedInterpretation, len(list.Interpretations))
	for id, interpretation := range list.Interpretations {
		requirements, err := encodeRequirements(interpretation.Requirements)
		if err != nil {
			return protectedList{}, err
		}
		interpretations[id] = protectedInterpretation{
			Name:         interpretation.Name,
			Description:  interpretation.Description,
			Requirements: requirements,
		}
	}

	dependencies := make(map[string]protectedDependency, len(list.Dependencies))
	for alias, dependency := range list.Dependencies {
		dependencies[alias] = protectedDependency{
			Owner:   dependency.Owner,
			Name:    dependency.Name,
			Version: dependency.Version,
		}
	}
	if len(dependencies) == 0 {
		dependencies = nil
	}

	return protectedList{
		Owner:           list.Owner,
		Name:            list.Name,
		Description:     list.Description,
		Version:         list.Version,
		IssuedAt:        formatTime(list.IssuedAt),
		Interpretations: interpretations,
		Dependencies:    dependencies,
	}, nil
}

func dependencyWireValue(dependency topic.Dependency, longForm, locationsPresent bool) any {
	if !longForm && len(dependency.Locations) == 0 {
		return fmt.Sprintf("%s/%s@%s", dependency.Owner, dependency.Name, dependency.Version)
	}

	var locations *[]string
	if locationsPresent || len(dependency.Locations) != 0 {
		copied := append([]string(nil), dependency.Locations...)
		locations = &copied
	}
	return wireDependency{
		Owner:     dependency.Owner,
		Name:      dependency.Name,
		Version:   dependency.Version,
		Locations: locations,
	}
}

func decodeJSON(data []byte) (List, error) {
	var raw rawList
	if err := decodeStrictJSON(data, &raw); err != nil {
		return List{}, fmt.Errorf("unmarshal score interpretation list JSON: %w", err)
	}

	issuedAt, err := time.Parse(time.RFC3339, raw.IssuedAt)
	if err != nil {
		return List{}, fmt.Errorf("unmarshal score interpretation list JSON: decode issued-at: %w", err)
	}
	decoded := List{
		Owner:                      raw.Owner,
		Name:                       raw.Name,
		Description:                raw.Description,
		Version:                    raw.Version,
		IssuedAt:                   issuedAt,
		Interpretations:            make(map[string]Interpretation, len(raw.Interpretations)),
		Dependencies:               make(map[string]topic.Dependency, len(raw.Dependencies)),
		schemaURL:                  raw.Schema,
		signature:                  raw.Signature,
		signedBy:                   raw.SignedBy,
		dependencyLongForm:         make(map[string]bool),
		dependencyLocationsPresent: make(map[string]bool),
	}

	for id, rawValue := range raw.Interpretations {
		interpretation, decodeErr := decodeInterpretation(rawValue, id)
		if decodeErr != nil {
			return List{}, fmt.Errorf("decode score interpretation %q: %w", id, decodeErr)
		}
		decoded.Interpretations[id] = interpretation
	}
	for alias, rawValue := range raw.Dependencies {
		dependency, decodeErr := decodeDependency(rawValue)
		if decodeErr != nil {
			return List{}, fmt.Errorf("decode dependency %q: %w", alias, decodeErr)
		}
		decoded.Dependencies[alias] = dependency.value
		decoded.dependencyLongForm[alias] = dependency.longForm
		decoded.dependencyLocationsPresent[alias] = dependency.locationsPresent
	}

	return decoded, nil
}

func decodeInterpretation(data json.RawMessage, mapID string) (Interpretation, error) {
	var raw rawInterpretation
	if err := decodeStrictJSON(data, &raw); err != nil {
		return Interpretation{}, err
	}
	id := raw.ID
	if id == "" {
		id = mapID
	}
	if id != mapID {
		return Interpretation{}, fmt.Errorf("map key %q does not match ID %q", mapID, id)
	}
	requirements, err := decodeRequirements(raw.Requirements)
	if err != nil {
		return Interpretation{}, err
	}
	return Interpretation{
		ID:           id,
		Name:         raw.Name,
		Description:  raw.Description,
		Requirements: requirements,
		schemaURL:    raw.Schema,
	}, nil
}

func decodeRequirements(data json.RawMessage) ([]Requirement, error) {
	if len(data) == 0 {
		return nil, errors.New("requirements are required")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode requirements: %w", err)
	}
	requirements := make([]Requirement, 0, len(raw))
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := raw[key]
		requirement, err := decodeRequirement(key, value)
		if err != nil {
			return nil, err
		}
		requirements = append(requirements, requirement)
	}
	return requirements, nil
}

func decodeRequirement(key string, value json.RawMessage) (Requirement, error) {
	if strings.Contains(key, ".") {
		dependency, topicID, found := strings.Cut(key, ".")
		if !found || strings.Contains(topicID, ".") {
			return nil, fmt.Errorf("invalid qualified topic %q", key)
		}
		var minimum score.Score
		if err := json.Unmarshal(value, &minimum); err != nil {
			return nil, fmt.Errorf("decode requirement %q: %w", key, err)
		}
		return Require(dependency, topicID, minimum), nil
	}

	children, err := decodeRequirements(value)
	if err != nil {
		return nil, fmt.Errorf("decode operator %q: %w", key, err)
	}
	switch {
	case allOperatorPattern.MatchString(key):
		return All{ID: key, Requirements: children}, nil
	case anyOperatorPattern.MatchString(key):
		return Any{ID: key, Requirements: children}, nil
	case atLeastOperatorPattern.MatchString(key):
		count, countErr := parseAtLeastCount(key)
		if countErr != nil {
			return nil, countErr
		}
		return AtLeast{ID: key, MinCount: count, Requirements: children}, nil
	default:
		return nil, fmt.Errorf("unknown requirement operator %q", key)
	}
}

type decodedDependency struct {
	value            topic.Dependency
	longForm         bool
	locationsPresent bool
}

func decodeDependency(data json.RawMessage) (decodedDependency, error) {
	var shorthand string
	if err := json.Unmarshal(data, &shorthand); err == nil {
		at := strings.LastIndexByte(shorthand, '@')
		if at < 0 {
			return decodedDependency{}, fmt.Errorf("invalid dependency shorthand %q", shorthand)
		}
		ownerAndName := shorthand[:at]
		owner, name, found := strings.Cut(ownerAndName, "/")
		if !found || strings.Contains(name, "/") {
			return decodedDependency{}, fmt.Errorf("invalid dependency shorthand %q", shorthand)
		}
		return decodedDependency{
			value: topic.Dependency{
				Owner:   owner,
				Name:    name,
				Version: shorthand[at+1:],
			},
		}, nil
	}

	var raw rawDependency
	if err := decodeStrictJSON(data, &raw); err != nil {
		return decodedDependency{}, err
	}
	dependency := topic.Dependency{
		Owner:   raw.Owner,
		Name:    raw.Name,
		Version: raw.Version,
	}
	if raw.Locations != nil {
		dependency.Locations = append([]string(nil), (*raw.Locations)...)
	}
	return decodedDependency{
		value:            dependency,
		longForm:         true,
		locationsPresent: raw.Locations != nil,
	}, nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return ensureJSONEOF(decoder)
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
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func ensureYAMLEOF(decoder *yaml.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple YAML documents are not allowed")
		}
		return err
	}
	return nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
