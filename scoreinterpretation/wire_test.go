package scoreinterpretation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/topic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListMarshalJSONEncodesNestedRequirementsAndDependencies(t *testing.T) {
	// Description
	// Exported operator IDs resolve to schema-valid keys and dependencies use topic wire forms.

	// Arrange
	list := validInterpretationList()
	list.Interpretations = map[string]Interpretation{
		"nested": {
			ID:          "nested",
			Name:        "Nested",
			Description: "Nested logical requirements.",
			Requirements: []Requirement{
				All{
					ID: "core",
					Requirements: []Requirement{
						Require("math", "addition", score.Competent),
					},
				},
				Any{
					ID: "any-advanced",
					Requirements: []Requirement{
						Require("math", "calculus", score.Familiar),
					},
				},
				AtLeast{
					ID:       "pedagogy",
					MinCount: 2,
					Requirements: []Requirement{
						Require("pedagogy", "lesson-planning", score.Competent),
						Require("pedagogy", "classroom-management", score.Competent),
					},
				},
			},
		},
	}
	list.Dependencies["pedagogy"] = topic.Dependency{
		Owner:     "example.com",
		Name:      "pedagogy",
		Version:   "0.1.0",
		Locations: []string{"https://example.com/pedagogy.yml"},
	}

	// Act
	data, err := list.MarshalJSON()

	// Assert
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(data, &document))
	interpretations := document["score-interpretations"].(map[string]any)
	nested := interpretations["nested"].(map[string]any)
	requirements := nested["requirements"].(map[string]any)
	assert.Contains(t, requirements, "all-core")
	assert.Contains(t, requirements, "any-advanced")
	assert.Contains(t, requirements, "at-least-2-pedagogy")
	assert.Equal(t, "example.com/math@0.1.0", document["dependencies"].(map[string]any)["math"])
	pedagogy := document["dependencies"].(map[string]any)["pedagogy"].(map[string]any)
	assert.Equal(t, "example.com", pedagogy["topic-list-owner"])
	assert.Equal(t, []any{"https://example.com/pedagogy.yml"}, pedagogy["locations"])
}

func TestListUnmarshalJSONSupportsPlainSchemaOperators(t *testing.T) {
	// Description
	// Plain all, any, and at-least-N operators remain readable and round-trip unchanged.

	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"plain-operators",
		"description":"Plain schema operators.",
		"version":"0.1.0",
		"issued-at":"2026-01-26T01:00:00Z",
		"signature":null,
		"signed-by":null,
		"score-interpretations":{
			"plain":{
				"name":"Plain",
				"description":"Plain operators.",
				"requirements":{
					"all":{"math.addition":"competent"},
					"any":{"math.calculus":"familiar"},
					"at-least-1":{"math.subtraction":"aware"}
				}
			}
		},
		"dependencies":{"math":"example.com/math@0.1.0"}
	}`)
	var list List

	// Act
	err := list.UnmarshalJSON(data)
	roundTrip, marshalErr := list.MarshalJSON()

	// Assert
	require.NoError(t, err)
	require.NoError(t, marshalErr)
	assert.Contains(t, string(roundTrip), `"all":`)
	assert.Contains(t, string(roundTrip), `"any":`)
	assert.Contains(t, string(roundTrip), `"at-least-1":`)
	assert.Equal(t, "all", list.Interpretations["plain"].Requirements[0].(All).ID)
}

func TestListYAMLRoundTripPreservesNestedContent(t *testing.T) {
	// Description
	// Direct YAML byte APIs preserve nested logical requirements and dependency locations.

	// Arrange
	original := validInterpretationList()
	original.Dependencies["math"] = topic.Dependency{
		Owner:     "example.com",
		Name:      "math",
		Version:   "0.1.0",
		Locations: []string{"https://example.com/math.yml", "npm:@example/math@0.1.0"},
	}
	var decoded List

	// Act
	data, marshalErr := original.MarshalYAML()
	unmarshalErr := decoded.UnmarshalYAML(data)
	roundTrip, roundTripErr := decoded.MarshalYAML()

	// Assert
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	require.NoError(t, roundTripErr)
	assert.Equal(t, original.Owner, decoded.Owner)
	assert.Equal(t, original.Interpretations["arithmetic"].Name, decoded.Interpretations["arithmetic"].Name)
	assert.Equal(t, original.Dependencies["math"].Locations, decoded.Dependencies["math"].Locations)
	assert.Contains(t, string(roundTrip), "any-advanced:")
	assert.Contains(t, string(roundTrip), "at-least-2-pedagogy:")
}

func TestListJSONRoundTripPreservesLongDependencyWithoutLocations(t *testing.T) {
	// Description
	// A decoded long-form dependency does not collapse to shorthand when locations are absent.

	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"long-dependency",
		"description":"Long dependency form.",
		"version":"0.1.0",
		"issued-at":"2026-01-26T01:00:00Z",
		"signature":null,
		"signed-by":null,
		"score-interpretations":{
			"arithmetic":{
				"name":"Arithmetic",
				"description":"Arithmetic.",
				"requirements":{"math.addition":"competent"}
			}
		},
		"dependencies":{
			"math":{
				"topic-list-owner":"example.com",
				"topic-list-name":"math",
				"topic-list-version":"0.1.0"
			}
		}
	}`)
	var list List

	// Act
	err := list.UnmarshalJSON(data)
	roundTrip, marshalErr := list.MarshalJSON()

	// Assert
	require.NoError(t, err)
	require.NoError(t, marshalErr)
	assert.Contains(t, string(roundTrip), `"math":{"topic-list-owner":"example.com"`)
}

func TestListUnmarshalJSONRejectsStaleURLDependencyShape(t *testing.T) {
	// Description
	// Interpretation lists use topic.Dependency shorthand rather than the stale URI-only list schema.

	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"bad-dependency",
		"description":"Bad dependency.",
		"version":"0.1.0",
		"issued-at":"2026-01-26T01:00:00Z",
		"signature":null,
		"signed-by":null,
		"score-interpretations":{},
		"dependencies":{"math":"https://example.com/math.yml"}
	}`)
	var list List

	// Act
	err := list.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dependency shorthand")
}

func TestListUnmarshalJSONRejectsUnknownField(t *testing.T) {
	// Description
	// Unknown wire fields are rejected instead of being silently discarded.

	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"unknown-field",
		"description":"Unknown field.",
		"version":"0.1.0",
		"issued-at":"2026-01-26T01:00:00Z",
		"signature":null,
		"signed-by":null,
		"score-interpretations":{},
		"unexpected":true
	}`)
	var list List

	// Act
	err := list.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "additional properties")
}

func TestListUnmarshalJSONStillValidatesNonDependencySchemaFields(t *testing.T) {
	// Description
	// The dependency-shape bypass does not weaken signature or other list schema validation.

	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"invalid-signature",
		"description":"Invalid signature.",
		"version":"0.1.0",
		"issued-at":"2026-01-26T01:00:00Z",
		"signature":"not-armored",
		"signed-by":"proficiency@example.com",
		"score-interpretations":{},
		"dependencies":{"math":"example.com/math@0.1.0"}
	}`)
	var list List

	// Act
	err := list.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature")
}

func validInterpretationList() List {
	return List{
		Owner:       "example.com",
		Name:        "math-pathways",
		Description: "Mathematics proficiency levels with logical composition.",
		Version:     "0.1.0",
		IssuedAt:    time.Date(2026, time.January, 26, 1, 0, 0, 0, time.UTC),
		Dependencies: map[string]topic.Dependency{
			"math": {
				Owner:   "example.com",
				Name:    "math",
				Version: "0.1.0",
			},
			"pedagogy": {
				Owner:   "example.com",
				Name:    "pedagogy",
				Version: "0.1.0",
			},
		},
		Interpretations: map[string]Interpretation{
			"arithmetic": {
				ID:          "arithmetic",
				Name:        "Arithmetic",
				Description: "Arithmetic and related skills.",
				Requirements: []Requirement{
					Require("math", "addition", score.Competent),
					Any{
						ID: "any-advanced",
						Requirements: []Requirement{
							Require("math", "trigonometry", score.Competent),
							Require("math", "calculus", score.Competent),
						},
					},
					AtLeast{
						ID:       "at-least-2-pedagogy",
						MinCount: 2,
						Requirements: []Requirement{
							Require("pedagogy", "lesson-planning", score.Competent),
							Require("pedagogy", "classroom-management", score.Competent),
						},
					},
				},
			},
		},
	}
}
