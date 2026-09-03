package canonical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONOrdersMapKeys(t *testing.T) {
	// Description
	// Equivalent maps produce identical compact JSON regardless of insertion order.

	// Arrange
	first := map[string]any{
		"z": 1,
		"a": "<value>",
	}
	second := map[string]any{
		"a": "<value>",
		"z": 1,
	}
	expected := `{"a":"<value>","z":1}`

	// Act
	firstJSON, firstErr := JSON(first)
	secondJSON, secondErr := JSON(second)

	// Assert - Encoding
	require.NoError(t, firstErr)
	require.NoError(t, secondErr)

	// Assert - Determinism
	assert.Equal(t, expected, string(firstJSON))
	assert.Equal(t, firstJSON, secondJSON)
}

func TestStateDetectsProtectedContentChange(t *testing.T) {
	// Description
	// Captured protected state becomes stale after protected content changes.

	// Arrange
	original := map[string]string{
		"name": "math",
	}
	changed := map[string]string{
		"name": "science",
	}
	state, stateErr := NewState(original)
	require.NoError(t, stateErr)

	// Act
	originalMatches, originalErr := state.Matches(original)
	changedMatches, changedErr := state.Matches(changed)

	// Assert - Matching
	require.NoError(t, originalErr)
	assert.True(t, originalMatches)

	// Assert - Staleness
	require.NoError(t, changedErr)
	assert.False(t, changedMatches)
}

func TestZeroStateIsNotInitialized(t *testing.T) {
	// Description
	// A zero state reports that no protected content has been captured.

	// Arrange
	var state State
	value := "content"

	// Act
	initialized := state.Initialized()
	matches, err := state.Matches(value)

	// Assert
	require.NoError(t, err)
	assert.False(t, initialized)
	assert.False(t, matches)
}
