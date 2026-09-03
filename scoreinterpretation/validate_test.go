package scoreinterpretation

import (
	"testing"

	"github.com/openproficiency/opm-go/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListValidateRejectsQualifiedTopicReuseAcrossInterpretations(t *testing.T) {
	// Description
	// A qualified topic may appear only once across all interpretations in a list.

	// Arrange
	list := validInterpretationList()
	list.Interpretations["advanced"] = Interpretation{
		ID:          "advanced",
		Name:        "Advanced",
		Description: "Another interpretation using addition.",
		Requirements: []Requirement{
			Require("math", "addition", score.Fluent),
		},
	}

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate qualified topic "math.addition"`)
}

func TestInterpretationValidateRejectsDuplicateQualifiedTopic(t *testing.T) {
	// Description
	// A qualified topic may appear only once across one interpretation tree.

	// Arrange
	interpretation := Interpretation{
		ID:          "duplicate-topic",
		Name:        "Duplicate",
		Description: "Duplicate topic.",
		Requirements: []Requirement{
			Require("math", "addition", score.Competent),
			All{
				ID: "nested",
				Requirements: []Requirement{
					Require("math", "addition", score.Fluent),
				},
			},
		},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate qualified topic "math.addition"`)
}

func TestInterpretationValidateRejectsDuplicateOperatorID(t *testing.T) {
	// Description
	// Resolved operator IDs are unique among siblings in one requirements layer.

	// Arrange
	interpretation := Interpretation{
		ID:          "duplicate-operator",
		Name:        "Duplicate",
		Description: "Duplicate operator.",
		Requirements: []Requirement{
			Any{ID: "pathway", Requirements: []Requirement{Require("math", "addition", score.Aware)}},
			Any{ID: "any-pathway", Requirements: []Requirement{Require("math", "subtraction", score.Aware)}},
		},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate operator ID "any-pathway"`)
}

func TestInterpretationValidateAllowsSameOperatorIDAtDifferentLevels(t *testing.T) {
	// Description
	// The same resolved operator key may be reused at a nested requirements layer.

	// Arrange
	interpretation := Interpretation{
		ID:          "nested-operator-reuse",
		Name:        "Nested Operator Reuse",
		Description: "Operator IDs are scoped to their immediate layer.",
		Requirements: []Requirement{
			All{
				ID: "shared",
				Requirements: []Requirement{
					All{
						ID: "shared",
						Requirements: []Requirement{
							Require("math", "addition", score.Aware),
						},
					},
				},
			},
		},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.NoError(t, err)
}

func TestInterpretationValidateAllowsSameOperatorIDInSeparateGroups(t *testing.T) {
	// Description
	// Separate parent groups may each contain a child with the same resolved operator key.

	// Arrange
	interpretation := Interpretation{
		ID:          "group-operator-reuse",
		Name:        "Group Operator Reuse",
		Description: "Operator IDs are scoped independently in separate groups.",
		Requirements: []Requirement{
			All{
				ID: "first-parent",
				Requirements: []Requirement{
					Any{
						ID: "shared-child",
						Requirements: []Requirement{
							Require("math", "addition", score.Aware),
						},
					},
				},
			},
			All{
				ID: "second-parent",
				Requirements: []Requirement{
					Any{
						ID: "shared-child",
						Requirements: []Requirement{
							Require("math", "subtraction", score.Aware),
						},
					},
				},
			},
		},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.NoError(t, err)
}

func TestInterpretationValidateRejectsMissingOperatorID(t *testing.T) {
	// Description
	// Programmatically constructed logical operators require an explicit ID.

	// Arrange
	interpretation := Interpretation{
		ID:          "missing-operator",
		Name:        "Missing",
		Description: "Missing operator ID.",
		Requirements: []Requirement{
			Any{Requirements: []Requirement{Require("math", "addition", score.Aware)}},
		},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "any operator ID is required")
}

func TestInterpretationValidateRejectsAtLeastCountConflict(t *testing.T) {
	// Description
	// A full at-least operator ID cannot contradict MinCount.

	// Arrange
	interpretation := Interpretation{
		ID:          "count-conflict",
		Name:        "Conflict",
		Description: "Conflicting minimum.",
		Requirements: []Requirement{
			AtLeast{
				ID:           "at-least-3-pathway",
				MinCount:     2,
				Requirements: []Requirement{Require("math", "addition", score.Aware)},
			},
		},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts with minimum count 2")
}

func TestListValidateRejectsUnknownDependencyAlias(t *testing.T) {
	// Description
	// Every leaf dependency alias must be declared by the list.

	// Arrange
	list := validInterpretationList()
	list.Interpretations["arithmetic"] = Interpretation{
		ID:          "arithmetic",
		Name:        "Arithmetic",
		Description: "Unknown dependency.",
		Requirements: []Requirement{
			Require("unknown", "addition", score.Competent),
		},
	}

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown dependency alias "unknown"`)
}

func TestListValidateRejectsInterpretationMapKeyMismatch(t *testing.T) {
	// Description
	// Interpretation map keys must exactly match exported interpretation IDs.

	// Arrange
	list := validInterpretationList()
	interpretation := list.Interpretations["arithmetic"]
	interpretation.ID = "different"
	list.Interpretations["arithmetic"] = interpretation

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match interpretation ID")
}

func TestInterpretationValidateRejectsInvalidID(t *testing.T) {
	// Description
	// Interpretation IDs follow the schema kebab-case constraint.

	// Arrange
	interpretation := Interpretation{
		ID:           "Math Tutor",
		Name:         "Math Tutor",
		Description:  "Invalid ID.",
		Requirements: []Requirement{},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid kebab-case")
}
