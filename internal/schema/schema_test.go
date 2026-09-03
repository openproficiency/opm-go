package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedSchemasCompile(t *testing.T) {
	// Description
	// Every bundled v0.1.1 schema compiles without retrieving remote references.

	// Arrange
	scoreInterpretationListName := ScoreInterpretationList
	scoreInterpretationName := ScoreInterpretation
	topicListName := TopicList
	topicName := Topic
	transcriptEntryVerificationName := TranscriptEntryVerification
	transcriptEntryName := TranscriptEntry
	transcriptName := Transcript

	// Act
	scoreInterpretationListSchema, scoreInterpretationListErr := Compile(scoreInterpretationListName)
	scoreInterpretationSchema, scoreInterpretationErr := Compile(scoreInterpretationName)
	topicListSchema, topicListErr := Compile(topicListName)
	topicSchema, topicErr := Compile(topicName)
	transcriptEntryVerificationSchema, transcriptEntryVerificationErr := Compile(transcriptEntryVerificationName)
	transcriptEntrySchema, transcriptEntryErr := Compile(transcriptEntryName)
	transcriptSchema, transcriptErr := Compile(transcriptName)

	// Assert - Compilation
	require.NoError(t, scoreInterpretationListErr)
	require.NoError(t, scoreInterpretationErr)
	require.NoError(t, topicListErr)
	require.NoError(t, topicErr)
	require.NoError(t, transcriptEntryVerificationErr)
	require.NoError(t, transcriptEntryErr)
	require.NoError(t, transcriptErr)

	// Assert - Results
	assert.NotNil(t, scoreInterpretationListSchema)
	assert.NotNil(t, scoreInterpretationSchema)
	assert.NotNil(t, topicListSchema)
	assert.NotNil(t, topicSchema)
	assert.NotNil(t, transcriptEntryVerificationSchema)
	assert.NotNil(t, transcriptEntrySchema)
	assert.NotNil(t, transcriptSchema)
}

func TestTopicListValidatesEmbeddedRemoteReference(t *testing.T) {
	// Description
	// A topic list validates its nested topic through the locally registered main-branch URL.

	// Arrange
	name := TopicList
	document := []byte(`{
		"owner": "example.com",
		"name": "math",
		"description": "Basic mathematics",
		"version": "0.1.0",
		"issued-at": "2026-09-01T00:00:00Z",
		"signature": null,
		"signed-by": null,
		"topics": {
			"addition": {
				"description": "Combining quantities"
			}
		}
	}`)

	// Act
	err := ValidateJSON(name, document)

	// Assert
	require.NoError(t, err)
}

func TestTopicListRejectsInvalidNestedTopic(t *testing.T) {
	// Description
	// Nested topic constraints remain active when a remote reference is resolved offline.

	// Arrange
	name := TopicList
	document := []byte(`{
		"owner": "example.com",
		"name": "math",
		"description": "Basic mathematics",
		"version": "0.1.0",
		"issued-at": "2026-09-01T00:00:00Z",
		"signature": null,
		"signed-by": null,
		"topics": {
			"addition": {
				"description": "Combining quantities",
				"unexpected": true
			}
		}
	}`)

	// Act
	err := ValidateJSON(name, document)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected")
}

func TestTopicListValidatesYAML(t *testing.T) {
	// Description
	// YAML documents are normalized and checked against the same embedded schema.

	// Arrange
	name := TopicList
	document := []byte(`owner: example.com
name: math
description: Basic mathematics
version: 0.1.0
issued-at: 2026-09-01T00:00:00Z
signature: null
signed-by: null
topics:
  addition:
    description: Combining quantities
`)

	// Act
	err := ValidateYAML(name, document)

	// Assert
	require.NoError(t, err)
}

func TestUnknownSchemaReturnsError(t *testing.T) {
	// Description
	// Callers receive an error instead of a nil schema for an unknown name.

	// Arrange
	name := Name("missing.schema.json")

	// Act
	compiledSchema, err := Compile(name)

	// Assert
	require.Error(t, err)
	assert.Nil(t, compiledSchema)
	assert.Contains(t, err.Error(), "unknown OPM schema")
}
