package score_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/openproficiency/opm-go/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestScoreOrdering(t *testing.T) {
	// Arrange
	unaware := score.Unaware
	aware := score.Aware
	familiar := score.Familiar
	competent := score.Competent
	fluent := score.Fluent

	// Act
	unawareBeforeAware := unaware < aware
	awareBeforeFamiliar := aware < familiar
	familiarBeforeCompetent := familiar < competent
	competentBeforeFluent := competent < fluent

	// Assert
	assert.True(t, unawareBeforeAware)
	assert.True(t, awareBeforeFamiliar)
	assert.True(t, familiarBeforeCompetent)
	assert.True(t, competentBeforeFluent)
}

func TestScoreStringReturnsCanonicalNames(t *testing.T) {
	// Arrange
	unaware := score.Unaware
	aware := score.Aware
	familiar := score.Familiar
	competent := score.Competent
	fluent := score.Fluent

	// Act
	unawareName := unaware.String()
	awareName := aware.String()
	familiarName := familiar.String()
	competentName := competent.String()
	fluentName := fluent.String()

	// Assert
	assert.Equal(t, "unaware", unawareName)
	assert.Equal(t, "aware", awareName)
	assert.Equal(t, "familiar", familiarName)
	assert.Equal(t, "competent", competentName)
	assert.Equal(t, "fluent", fluentName)
}

func TestScoreJSONRoundTrip(t *testing.T) {
	// Arrange
	original := score.Competent

	// Act
	data, marshalErr := json.Marshal(original)
	var decoded score.Score
	unmarshalErr := json.Unmarshal(data, &decoded)

	// Assert
	require.NoError(t, marshalErr)
	assert.Equal(t, `"competent"`, string(data))
	require.NoError(t, unmarshalErr)
	assert.Equal(t, original, decoded)
}

func TestScoreJSONUsesEveryCanonicalValue(t *testing.T) {
	// Arrange
	unaware := score.Unaware
	aware := score.Aware
	familiar := score.Familiar
	competent := score.Competent
	fluent := score.Fluent

	// Act
	unawareJSON, unawareErr := json.Marshal(unaware)
	awareJSON, awareErr := json.Marshal(aware)
	familiarJSON, familiarErr := json.Marshal(familiar)
	competentJSON, competentErr := json.Marshal(competent)
	fluentJSON, fluentErr := json.Marshal(fluent)

	// Assert
	require.NoError(t, unawareErr)
	assert.Equal(t, `"unaware"`, string(unawareJSON))
	require.NoError(t, awareErr)
	assert.Equal(t, `"aware"`, string(awareJSON))
	require.NoError(t, familiarErr)
	assert.Equal(t, `"familiar"`, string(familiarJSON))
	require.NoError(t, competentErr)
	assert.Equal(t, `"competent"`, string(competentJSON))
	require.NoError(t, fluentErr)
	assert.Equal(t, `"fluent"`, string(fluentJSON))
}

func TestScoreJSONRejectsUnknownValue(t *testing.T) {
	// Arrange
	data := []byte(`"expert"`)
	var decoded score.Score

	// Act
	err := json.Unmarshal(data, &decoded)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, score.ErrInvalidScore))
}

func TestScoreJSONRejectsWrongCase(t *testing.T) {
	// Arrange
	data := []byte(`"Fluent"`)
	var decoded score.Score

	// Act
	err := json.Unmarshal(data, &decoded)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, score.ErrInvalidScore))
}

func TestScoreJSONRejectsNonString(t *testing.T) {
	// Arrange
	data := []byte(`4`)
	var decoded score.Score

	// Act
	err := json.Unmarshal(data, &decoded)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, score.ErrInvalidScore))
}

func TestInvalidScoreJSONMarshalFails(t *testing.T) {
	// Arrange
	invalid := score.Score(5)

	// Act
	data, err := json.Marshal(invalid)

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.True(t, errors.Is(err, score.ErrInvalidScore))
}

func TestNegativeScoreJSONMarshalFails(t *testing.T) {
	// Arrange
	invalid := score.Score(-1)

	// Act
	data, err := json.Marshal(invalid)

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.True(t, errors.Is(err, score.ErrInvalidScore))
}

func TestScoreYAMLRoundTrip(t *testing.T) {
	// Arrange
	original := score.Familiar

	// Act
	data, marshalErr := yaml.Marshal(original)
	var decoded score.Score
	unmarshalErr := yaml.Unmarshal(data, &decoded)

	// Assert
	require.NoError(t, marshalErr)
	assert.Equal(t, "familiar\n", string(data))
	require.NoError(t, unmarshalErr)
	assert.Equal(t, original, decoded)
}

func TestScoreYAMLUsesEveryCanonicalValue(t *testing.T) {
	// Arrange
	unaware := score.Unaware
	aware := score.Aware
	familiar := score.Familiar
	competent := score.Competent
	fluent := score.Fluent

	// Act
	unawareYAML, unawareErr := yaml.Marshal(unaware)
	awareYAML, awareErr := yaml.Marshal(aware)
	familiarYAML, familiarErr := yaml.Marshal(familiar)
	competentYAML, competentErr := yaml.Marshal(competent)
	fluentYAML, fluentErr := yaml.Marshal(fluent)

	// Assert
	require.NoError(t, unawareErr)
	assert.Equal(t, "unaware\n", string(unawareYAML))
	require.NoError(t, awareErr)
	assert.Equal(t, "aware\n", string(awareYAML))
	require.NoError(t, familiarErr)
	assert.Equal(t, "familiar\n", string(familiarYAML))
	require.NoError(t, competentErr)
	assert.Equal(t, "competent\n", string(competentYAML))
	require.NoError(t, fluentErr)
	assert.Equal(t, "fluent\n", string(fluentYAML))
}

func TestScoreYAMLRejectsUnknownValue(t *testing.T) {
	// Arrange
	data := []byte("expert\n")
	var decoded score.Score

	// Act
	err := yaml.Unmarshal(data, &decoded)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, score.ErrInvalidScore))
}

func TestScoreYAMLRejectsNonString(t *testing.T) {
	// Arrange
	data := []byte("4\n")
	var decoded score.Score

	// Act
	err := yaml.Unmarshal(data, &decoded)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, score.ErrInvalidScore))
}

func TestInvalidScoreYAMLMarshalFails(t *testing.T) {
	// Arrange
	invalid := score.Score(9)

	// Act
	data, err := yaml.Marshal(invalid)

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.True(t, errors.Is(err, score.ErrInvalidScore))
}
