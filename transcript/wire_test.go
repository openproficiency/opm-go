package transcript

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/openproficiency/opm-go/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntryMarshalJSONEncodesUnsignedMetadataAsNull(t *testing.T) {
	// Description
	// An unsigned entry remains storable with explicit null signing metadata.

	// Arrange
	entry := validTestEntry()
	expected := `{"user-email":"learner@example.com","topic":"addition","topic-list":"math","topic-list-version":"0.1.0","topic-list-owner":"example.com","score":"competent","issued-at":"2026-09-01T00:00:00Z","valid-until":"2028-09-01T00:00:00Z","issued-by":"example.com","signature":null,"signed-by":null}`

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.NoError(t, err)
	assert.JSONEq(t, expected, string(data))
	assert.Equal(t, expected, string(data))
}

func TestEntryUnmarshalJSONLoadsUnsignedMetadata(t *testing.T) {
	// Description
	// An unsigned schema-shaped entry decodes into the public fields.

	// Arrange
	data := []byte(`{
		"user-email":"learner@example.com",
		"topic":"addition",
		"topic-list":"math",
		"topic-list-version":"0.1.0",
		"topic-list-owner":"example.com",
		"score":"competent",
		"issued-at":"2026-09-01T00:00:00Z",
		"valid-until":"2028-09-01T00:00:00Z",
		"issued-by":"example.com",
		"signature":null,
		"signed-by":null
	}`)
	var entry Entry

	// Act
	err := entry.UnmarshalJSON(data)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, validTestEntry(), entry)
}

func TestEntryJSONRoundTripPreservesExtendedFieldsAndSchema(t *testing.T) {
	// Description
	// Every transcript-entry schema field survives JSON decoding and re-encoding.

	// Arrange
	data := []byte(`{
		"$schema":"https://raw.githubusercontent.com/openproficiency/model/refs/heads/main/schemas/transcript-entry.schema.json",
		"user-email":"learner@example.com",
		"topic":"addition",
		"topic-list":"math",
		"topic-list-version":"0.1.0",
		"topic-list-owner":"example.com",
		"topic-list-sources":[],
		"score":"competent",
		"issued-at":"2026-09-01T00:00:00Z",
		"valid-until":"2028-09-01T00:00:00Z",
		"issued-by":"example.com",
		"verification-url":"https://example.com/verify",
		"signature":null,
		"signed-by":null
	}`)
	var entry Entry

	// Act
	unmarshalErr := entry.UnmarshalJSON(data)
	encoded, marshalErr := entry.MarshalJSON()

	// Assert
	require.NoError(t, unmarshalErr)
	require.NoError(t, marshalErr)
	assert.JSONEq(t, string(data), string(encoded))
	assert.NotNil(t, entry.TopicListSources)
	assert.Empty(t, entry.TopicListSources)
	assert.Equal(t, "https://example.com/verify", entry.VerificationURL)
}

func TestEntryMarshalJSONNormalizesTimestampsToUTC(t *testing.T) {
	// Description
	// Wire timestamps use deterministic UTC RFC 3339 values.

	// Arrange
	entry := validTestEntry()
	offset := time.FixedZone("UTC-7", -7*60*60)
	entry.IssuedAt = time.Date(2026, time.August, 31, 17, 0, 0, 0, offset)
	entry.ValidUntil = time.Date(2028, time.August, 31, 17, 0, 0, 0, offset)

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.NoError(t, err)
	assert.Contains(t, string(data), `"issued-at":"2026-09-01T00:00:00Z"`)
	assert.Contains(t, string(data), `"valid-until":"2028-09-01T00:00:00Z"`)
}

func TestEntryUnmarshalJSONRejectsUnknownField(t *testing.T) {
	// Description
	// Additional properties are rejected as required by the OPM schema.

	// Arrange
	data := []byte(`{
		"user-email":"learner@example.com",
		"topic":"addition",
		"topic-list":"math",
		"topic-list-version":"0.1.0",
		"topic-list-owner":"example.com",
		"score":"competent",
		"issued-at":"2026-09-01T00:00:00Z",
		"valid-until":"2028-09-01T00:00:00Z",
		"issued-by":"example.com",
		"signature":null,
		"signed-by":null,
		"unexpected":true
	}`)
	var entry Entry

	// Act
	err := entry.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown field")
}

func TestEntryUnmarshalJSONRejectsPartialSignatureMetadata(t *testing.T) {
	// Description
	// Signature and signer metadata cannot be independently present.

	// Arrange
	data := []byte(`{
		"user-email":"learner@example.com",
		"topic":"addition",
		"topic-list":"math",
		"topic-list-version":"0.1.0",
		"topic-list-owner":"example.com",
		"score":"competent",
		"issued-at":"2026-09-01T00:00:00Z",
		"valid-until":"2028-09-01T00:00:00Z",
		"issued-by":"example.com",
		"signature":"signed",
		"signed-by":null
	}`)
	var entry Entry

	// Act
	err := entry.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "must both")
}

func TestEntryUnmarshalJSONRejectsInvalidScore(t *testing.T) {
	// Description
	// Scores outside the OPM enumeration are rejected during decoding.

	// Arrange
	data := []byte(`{
		"user-email":"learner@example.com",
		"topic":"addition",
		"topic-list":"math",
		"topic-list-version":"0.1.0",
		"topic-list-owner":"example.com",
		"score":"expert",
		"issued-at":"2026-09-01T00:00:00Z",
		"valid-until":"2028-09-01T00:00:00Z",
		"issued-by":"example.com",
		"signature":null,
		"signed-by":null
	}`)
	var entry Entry

	// Act
	err := entry.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid score")
}

func TestEntryUnmarshalJSONRejectsInvalidTimestamp(t *testing.T) {
	// Description
	// Date-time fields must use an RFC 3339 timestamp.

	// Arrange
	data := []byte(`{
		"user-email":"learner@example.com",
		"topic":"addition",
		"topic-list":"math",
		"topic-list-version":"0.1.0",
		"topic-list-owner":"example.com",
		"score":"competent",
		"issued-at":"September 1, 2026",
		"valid-until":"2028-09-01T00:00:00Z",
		"issued-by":"example.com",
		"signature":null,
		"signed-by":null
	}`)
	var entry Entry

	// Act
	err := entry.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "issued-at timestamp")
}

func TestEntryUnmarshalJSONRejectsMultipleValues(t *testing.T) {
	// Description
	// A transcript entry input contains exactly one JSON document.

	// Arrange
	data := []byte(`{} {}`)
	var entry Entry

	// Act
	err := entry.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "multiple JSON values")
}

func TestEntryMarshalYAMLEncodesUnsignedMetadataAsNull(t *testing.T) {
	// Description
	// YAML storage explicitly identifies an entry without a signature.

	// Arrange
	entry := validTestEntry()

	// Act
	data, err := entry.MarshalYAML()

	// Assert
	require.NoError(t, err)
	assert.Contains(t, string(data), "score: competent\n")
	assert.Contains(t, string(data), "signature: null\n")
	assert.Contains(t, string(data), "signed-by: null\n")
}

func TestEntryYAMLRoundTripPreservesExtendedFields(t *testing.T) {
	// Description
	// Extended mutable fields survive YAML decoding and encoding.

	// Arrange
	data := []byte(`$schema: https://raw.githubusercontent.com/openproficiency/model/refs/heads/main/schemas/transcript-entry.schema.json
user-email: learner@example.com
topic: addition
topic-list: math
topic-list-version: 0.1.0
topic-list-owner: example.com
topic-list-sources:
  - https://example.com/topic-lists/math/0.1.0
score: competent
issued-at: 2026-09-01T00:00:00Z
valid-until: 2028-09-01T00:00:00Z
issued-by: example.com
verification-url: https://example.com/verify
signature: null
signed-by: null
`)
	var entry Entry

	// Act
	unmarshalErr := entry.UnmarshalYAML(data)
	encoded, marshalErr := entry.MarshalYAML()

	// Assert
	require.NoError(t, unmarshalErr)
	require.NoError(t, marshalErr)
	assert.Contains(t, string(encoded), "$schema: https://raw.githubusercontent.com/openproficiency/model/refs/heads/main/schemas/transcript-entry.schema.json")
	assert.Equal(t, []string{"https://example.com/topic-lists/math/0.1.0"}, entry.TopicListSources)
	assert.Equal(t, "https://example.com/verify", entry.VerificationURL)
}

func TestEntryUnmarshalYAMLRejectsUnknownField(t *testing.T) {
	// Description
	// YAML additional properties receive the same strict handling as JSON.

	// Arrange
	data := []byte(`user-email: learner@example.com
topic: addition
topic-list: math
topic-list-version: 0.1.0
topic-list-owner: example.com
score: competent
issued-at: 2026-09-01T00:00:00Z
valid-until: 2028-09-01T00:00:00Z
issued-by: example.com
signature: null
signed-by: null
unexpected: true
`)
	var entry Entry

	// Act
	err := entry.UnmarshalYAML(data)

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown field")
}

func TestEntryUnmarshalYAMLRejectsMultipleDocuments(t *testing.T) {
	// Description
	// A transcript entry input cannot contain a second YAML document.

	// Arrange
	data := []byte(`user-email: learner@example.com
---
user-email: another@example.com
`)
	var entry Entry

	// Act
	err := entry.UnmarshalYAML(data)

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "multiple YAML documents")
}

func TestEntryStandardJSONEncodingUsesOPMWireShape(t *testing.T) {
	// Description
	// The standard JSON package invokes the entry's OPM encoding.

	// Arrange
	entry := validTestEntry()

	// Act
	data, err := json.Marshal(entry)

	// Assert
	require.NoError(t, err)
	assert.Contains(t, string(data), `"user-email":"learner@example.com"`)
	assert.NotContains(t, string(data), `"UserEmail"`)
}

func TestParseScoreLoadsEveryOPMLevel(t *testing.T) {
	// Description
	// The transcript wire decoder accepts all five assumed score values.

	// Arrange
	unaware := "unaware"
	aware := "aware"
	familiar := "familiar"
	competent := "competent"
	fluent := "fluent"

	// Act
	unawareScore, unawareErr := parseScore(unaware)
	awareScore, awareErr := parseScore(aware)
	familiarScore, familiarErr := parseScore(familiar)
	competentScore, competentErr := parseScore(competent)
	fluentScore, fluentErr := parseScore(fluent)

	// Assert
	require.NoError(t, unawareErr)
	require.NoError(t, awareErr)
	require.NoError(t, familiarErr)
	require.NoError(t, competentErr)
	require.NoError(t, fluentErr)
	assert.Equal(t, score.Unaware, unawareScore)
	assert.Equal(t, score.Aware, awareScore)
	assert.Equal(t, score.Familiar, familiarScore)
	assert.Equal(t, score.Competent, competentScore)
	assert.Equal(t, score.Fluent, fluentScore)
}
