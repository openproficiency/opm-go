package transcript

import (
	"testing"
	"time"

	"github.com/openproficiency/opm-go/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntryValidateRejectsInvalidUserEmail(t *testing.T) {
	// Description
	// Encoding rejects a user identifier that is not an email address.

	// Arrange
	entry := validTestEntry()
	entry.UserEmail = "learner"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "user email")
}

func TestEntryValidateRejectsInvalidTopic(t *testing.T) {
	// Description
	// Encoding rejects a topic identifier outside lowercase kebab case.

	// Arrange
	entry := validTestEntry()
	entry.Topic = "Long Division"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "topic")
}

func TestEntryValidateRejectsInvalidTopicList(t *testing.T) {
	// Description
	// Encoding rejects a topic-list name outside lowercase kebab case.

	// Arrange
	entry := validTestEntry()
	entry.TopicList = "Basic Math"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "topic list")
}

func TestEntryValidateRejectsInvalidTopicListVersion(t *testing.T) {
	// Description
	// Encoding rejects a topic-list version that is not semantic versioning.

	// Arrange
	entry := validTestEntry()
	entry.TopicListVersion = "version-one"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "topic list version")
}

func TestEntryValidateRejectsInvalidTopicListOwner(t *testing.T) {
	// Description
	// Encoding rejects a topic-list owner that is not an OPM hostname.

	// Arrange
	entry := validTestEntry()
	entry.TopicListOwner = "localhost"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "topic list owner")
}

func TestEntryValidateRejectsInvalidTopicListSource(t *testing.T) {
	// Description
	// Encoding rejects a topic-list source that is not an absolute URI.

	// Arrange
	entry := validTestEntry()
	entry.TopicListSources = []string{"topic-lists/math.yml"}

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "topic list source")
}

func TestEntryValidateRejectsInvalidScore(t *testing.T) {
	// Description
	// Encoding rejects a score outside the five OPM levels.

	// Arrange
	entry := validTestEntry()
	entry.Score = score.Score(99)

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "score")
}

func TestEntryValidateRejectsMissingIssuedAt(t *testing.T) {
	// Description
	// Encoding rejects an entry without an issuance timestamp.

	// Arrange
	entry := validTestEntry()
	entry.IssuedAt = time.Time{}

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "issued at")
}

func TestEntryValidateRejectsMissingValidUntil(t *testing.T) {
	// Description
	// Encoding rejects an entry without an expiration timestamp.

	// Arrange
	entry := validTestEntry()
	entry.ValidUntil = time.Time{}

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "valid until")
}

func TestEntryValidateRejectsInvalidIssuer(t *testing.T) {
	// Description
	// Encoding rejects an issuer that is not an OPM hostname.

	// Arrange
	entry := validTestEntry()
	entry.IssuedBy = "Example Incorporated"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "issuer")
}

func TestEntryValidateRejectsRelativeVerificationURL(t *testing.T) {
	// Description
	// Encoding rejects a verification endpoint without an absolute URI.

	// Arrange
	entry := validTestEntry()
	entry.VerificationURL = "/verify"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "verification URL")
}

func TestEntryValidateRejectsInsecureVerificationURL(t *testing.T) {
	// Description
	// Encoding enforces the specification's HTTPS verification endpoint requirement.

	// Arrange
	entry := validTestEntry()
	entry.VerificationURL = "http://example.com/verify"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorContains(t, err, "HTTPS")
}

func validTestEntry() Entry {
	return Entry{
		UserEmail:        "learner@example.com",
		Topic:            "addition",
		TopicList:        "math",
		TopicListVersion: "0.1.0",
		TopicListOwner:   "example.com",
		Score:            score.Competent,
		IssuedAt:         time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		ValidUntil:       time.Date(2028, time.September, 1, 0, 0, 0, 0, time.UTC),
		IssuedBy:         "example.com",
	}
}
