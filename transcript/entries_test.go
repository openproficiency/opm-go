package transcript

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/openproficiency/opm-go/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntriesJSONRoundTrip(t *testing.T) {
	// Description
	// A transcript collection preserves entry order and values in JSON.

	// Arrange
	first := validTestEntry()
	second := validTestEntry()
	second.Topic = "subtraction"
	second.Score = score.Fluent
	original := Entries{first, second}

	// Act
	data, marshalErr := original.MarshalJSON()
	var decoded Entries
	unmarshalErr := decoded.UnmarshalJSON(data)

	// Assert
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	assert.Equal(t, original, decoded)
}

func TestEntriesYAMLRoundTripMatchesREADMEStorage(t *testing.T) {
	// Description
	// The README's unsigned transcript list loads and remains portable.

	// Arrange
	data := []byte(`- user-email: learner@example.com
  topic-list-owner: example.com
  topic-list: math
  topic-list-version: 0.1.0
  topic: addition
  score: competent
  issued-at: 2026-09-01T00:00:00Z
  valid-until: 2028-09-01T00:00:00Z
  issued-by: example.com
- user-email: learner@example.com
  topic-list-owner: example.com
  topic-list: math
  topic-list-version: 0.1.0
  topic: subtraction
  score: competent
  issued-at: 2026-09-01T00:00:00Z
  valid-until: 2028-09-01T00:00:00Z
  issued-by: example.com
`)
	var entries Entries

	// Act
	unmarshalErr := entries.UnmarshalYAML(data)
	encoded, marshalErr := entries.MarshalYAML()

	// Assert
	require.NoError(t, unmarshalErr)
	require.NoError(t, marshalErr)
	require.Len(t, entries, 2)
	assert.Equal(t, "addition", entries[0].Topic)
	assert.Equal(t, "subtraction", entries[1].Topic)
	assert.Equal(t, 2, strings.Count(string(encoded), "signature: null"))
	assert.Equal(t, 2, strings.Count(string(encoded), "signed-by: null"))
}

func TestEntriesSignedJSONRoundTripVerifiesEachEntry(t *testing.T) {
	// Description
	// A fully signed JSON transcript conforms to the collection schema and remains verifiable.

	// Arrange
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	first := validTestEntry()
	second := validTestEntry()
	second.Topic = "subtraction"
	firstSignErr := first.Sign(entity, "")
	secondSignErr := second.Sign(entity, "")
	require.NoError(t, firstSignErr)
	require.NoError(t, secondSignErr)
	original := Entries{first, second}

	// Act
	data, marshalErr := original.MarshalJSON()
	var decoded Entries
	unmarshalErr := decoded.UnmarshalJSON(data)

	// Assert - Storage
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	require.Len(t, decoded, 2)

	// Act - Verification
	firstValid, firstVerifyErr := decoded[0].Verify(entity)
	secondValid, secondVerifyErr := decoded[1].Verify(entity)

	// Assert - Verification
	require.NoError(t, firstVerifyErr)
	require.NoError(t, secondVerifyErr)
	assert.True(t, firstValid)
	assert.True(t, secondValid)
}

func TestEntriesSignedYAMLRoundTripVerifiesEntry(t *testing.T) {
	// Description
	// A signed YAML transcript preserves armored signatures through collection storage.

	// Arrange
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	entry := validTestEntry()
	signErr := entry.Sign(entity, "")
	require.NoError(t, signErr)
	original := Entries{entry}

	// Act
	data, marshalErr := original.MarshalYAML()
	var decoded Entries
	unmarshalErr := decoded.UnmarshalYAML(data)

	// Assert - Storage
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	require.Len(t, decoded, 1)

	// Act - Verification
	valid, verifyErr := decoded[0].Verify(entity)

	// Assert - Verification
	require.NoError(t, verifyErr)
	assert.True(t, valid)
	assert.Contains(t, string(data), "-----BEGIN PGP SIGNATURE-----")
}

func TestEntriesMixedSignedAndUnsignedJSONRoundTrip(t *testing.T) {
	// Description
	// Portable storage may combine issued signatures with locally held unsigned entries.

	// Arrange
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	signed := validTestEntry()
	unsigned := validTestEntry()
	unsigned.Topic = "subtraction"
	signErr := signed.Sign(entity, "")
	require.NoError(t, signErr)
	original := Entries{signed, unsigned}

	// Act
	data, marshalErr := original.MarshalJSON()
	var decoded Entries
	unmarshalErr := decoded.UnmarshalJSON(data)

	// Assert - Storage
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	require.Len(t, decoded, 2)

	// Act - Verification
	signedValid, verifyErr := decoded[0].Verify(entity)

	// Assert - Verification
	require.NoError(t, verifyErr)
	assert.True(t, signedValid)
	assert.Equal(t, 1, strings.Count(string(data), `"signature":null`))
	assert.Empty(t, decoded[1].signature)
}

func TestEntriesStandardJSONEncodingUsesCollectionWireShape(t *testing.T) {
	// Description
	// The standard JSON package encodes Entries as an OPM array.

	// Arrange
	entries := Entries{validTestEntry()}

	// Act
	data, err := json.Marshal(entries)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, '[', rune(data[0]))
	assert.Contains(t, string(data), `"user-email":"learner@example.com"`)
}

func TestEntriesMarshalJSONRejectsEmptyCollection(t *testing.T) {
	// Description
	// A transcript must contain at least one entry.

	// Arrange
	entries := Entries{}

	// Act
	data, err := entries.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "at least one")
}

func TestEntriesUnmarshalJSONRejectsEmptyCollection(t *testing.T) {
	// Description
	// An empty JSON array is not a transcript under the OPM specification.

	// Arrange
	data := []byte(`[]`)
	var entries Entries

	// Act
	err := entries.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "at least one")
}

func TestEntriesUnmarshalJSONRejectsInvalidEntryWithIndex(t *testing.T) {
	// Description
	// Collection decoding identifies the invalid array element.

	// Arrange
	data := []byte(`[{
		"user-email":"invalid",
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
	}]`)
	var entries Entries

	// Act
	err := entries.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "entry 0")
}

func TestEntriesMarshalYAMLRejectsInvalidEntry(t *testing.T) {
	// Description
	// Collection encoding identifies which entry failed validation.

	// Arrange
	first := validTestEntry()
	second := validTestEntry()
	second.UserEmail = "invalid"
	entries := Entries{first, second}

	// Act
	data, err := entries.MarshalYAML()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "entry 1")
}

func TestEntriesUnmarshalYAMLRejectsNonCollection(t *testing.T) {
	// Description
	// A single YAML mapping cannot be decoded as a transcript collection.

	// Arrange
	data := []byte(`user-email: learner@example.com`)
	var entries Entries

	// Act
	err := entries.UnmarshalYAML(data)

	// Assert
	require.Error(t, err)
}
