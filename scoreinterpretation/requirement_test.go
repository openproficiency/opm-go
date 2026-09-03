package scoreinterpretation

import (
	"testing"

	"github.com/openproficiency/opm-go/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireCreatesSchemaQualifiedLeaf(t *testing.T) {
	// Description
	// Require creates the sealed leaf used by validation, encoding, and interpretation.

	// Arrange
	requirement := Require("std-math", "basic-math", score.Competent)

	// Act
	leaf, ok := requirement.(requiredTopic)

	// Assert
	require.True(t, ok)
	assert.Equal(t, "std-math.basic-math", leaf.key())
	assert.Equal(t, score.Competent, leaf.minimum)
}

func TestListAddInitializesAndReplacesByInterpretationID(t *testing.T) {
	// Description
	// Add initializes a nil map and replaces an existing interpretation with the same ID.

	// Arrange
	var list List
	first := Interpretation{ID: "arithmetic", Name: "First"}
	replacement := Interpretation{ID: "arithmetic", Name: "Replacement"}

	// Act
	list.Add(first)
	list.Add(replacement)

	// Assert
	require.Len(t, list.Interpretations, 1)
	assert.Equal(t, "Replacement", list.Interpretations["arithmetic"].Name)
}

func TestAllSuffixIDResolvesToSchemaOperatorKey(t *testing.T) {
	// Description
	// A readable exported suffix ID becomes a schema-valid all operator and result key.

	// Arrange
	operator := All{
		ID: "fundamentals",
		Requirements: []Requirement{
			Require("math", "addition", score.Aware),
		},
	}

	// Act
	key, value, err := encodeRequirement(operator)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "all-fundamentals", key)
	assert.Contains(t, value.(map[string]any), "math.addition")
}
