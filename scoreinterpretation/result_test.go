package scoreinterpretation

import (
	"testing"
	"time"

	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/topic"
	"github.com/openproficiency/opm-go/transcript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListInterpretReproducesDocumentedNestedResults(t *testing.T) {
	// Description
	// The documented tutor transcript produces the documented recursive booleans.

	// Arrange
	issuedAt := time.Date(2026, time.January, 26, 1, 0, 0, 0, time.UTC)
	list := List{
		Owner:       "example.com",
		Name:        "math-pathways",
		Description: "Mathematics proficiency levels with logical composition.",
		Version:     "0.1.0",
		IssuedAt:    issuedAt,
		Dependencies: map[string]topic.Dependency{
			"math": {
				Owner:   "example.com",
				Name:    "math",
				Version: "0.1.0",
			},
			"std-pedagogy": {
				Owner:   "example.com",
				Name:    "pedagogy",
				Version: "0.1.0",
			},
		},
		Interpretations: map[string]Interpretation{
			"math-tutor-1": {
				ID:          "math-tutor-1",
				Name:        "Math Tutor Level 1",
				Description: "Qualified to tutor students using arithmetic, advanced math, and pedagogy skills.",
				Requirements: []Requirement{
					Require("math", "addition", score.Competent),
					Require("math", "subtraction", score.Competent),
					Require("math", "multiplication", score.Competent),
					Require("math", "division", score.Competent),
					Require("std-pedagogy", "classroom-management", score.Competent),
					Any{
						ID: "any-advanced-subject",
						Requirements: []Requirement{
							Require("math", "trigonometry", score.Competent),
							Require("math", "calculus", score.Competent),
						},
					},
					AtLeast{
						ID:       "at-least-2-pedagogy-skills",
						MinCount: 2,
						Requirements: []Requirement{
							Require("std-pedagogy", "lesson-planning", score.Competent),
							Require("std-pedagogy", "lesson-customization", score.Competent),
							Require("std-pedagogy", "teacher-mentoring", score.Competent),
						},
					},
				},
			},
		},
	}
	entries := []transcript.Entry{
		testEntry("math", "addition", score.Competent, issuedAt),
		testEntry("math", "subtraction", score.Competent, issuedAt),
		testEntry("math", "multiplication", score.Competent, issuedAt),
		testEntry("math", "division", score.Competent, issuedAt),
		testEntry("math", "trigonometry", score.Familiar, issuedAt),
		testEntry("pedagogy", "classroom-management", score.Competent, issuedAt),
		testEntry("pedagogy", "lesson-planning", score.Competent, issuedAt),
	}

	// Act
	result, err := list.Interpret(entries, "math-tutor-1")

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.True(t, result.Requirements["math.addition"].Passed)
	assert.True(t, result.Requirements["std-pedagogy.classroom-management"].Passed)
	advanced := result.Requirements["any-advanced-subject"]
	assert.False(t, advanced.Passed)
	assert.False(t, advanced.Requirements["math.trigonometry"].Passed)
	assert.False(t, advanced.Requirements["math.calculus"].Passed)
	pedagogy := result.Requirements["at-least-2-pedagogy-skills"]
	assert.False(t, pedagogy.Passed)
	assert.True(t, pedagogy.Requirements["std-pedagogy.lesson-planning"].Passed)
	assert.False(t, pedagogy.Requirements["std-pedagogy.lesson-customization"].Passed)
}

func TestListInterpretMatchesOwnerNameVersionAndTopicExactly(t *testing.T) {
	// Description
	// A transcript score from a different topic-list version cannot satisfy a requirement.

	// Arrange
	list := validInterpretationList()
	entry := testEntry("math", "addition", score.Fluent, list.IssuedAt)
	entry.TopicListVersion = "0.2.0"

	// Act
	result, err := list.Interpret([]transcript.Entry{entry}, "arithmetic")

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Requirements["math.addition"].Passed)
}

func TestListInterpretEvaluatesAllAnyAndAtLeastPassingBranches(t *testing.T) {
	// Description
	// Explicit all, any, and at-least operators report passing recursive results.

	// Arrange
	list := validInterpretationList()
	list.Interpretations = map[string]Interpretation{
		"operators": {
			ID:          "operators",
			Name:        "Operators",
			Description: "Passing operators.",
			Requirements: []Requirement{
				All{
					ID: "fundamentals",
					Requirements: []Requirement{
						Require("math", "addition", score.Competent),
						Require("math", "subtraction", score.Aware),
					},
				},
				Any{
					ID: "advanced",
					Requirements: []Requirement{
						Require("math", "calculus", score.Competent),
						Require("math", "trigonometry", score.Competent),
					},
				},
				AtLeast{
					ID:       "breadth",
					MinCount: 2,
					Requirements: []Requirement{
						Require("pedagogy", "lesson-planning", score.Competent),
						Require("pedagogy", "classroom-management", score.Competent),
						Require("pedagogy", "teacher-mentoring", score.Competent),
					},
				},
			},
		},
	}
	entries := []transcript.Entry{
		testEntry("math", "addition", score.Fluent, list.IssuedAt),
		testEntry("math", "subtraction", score.Aware, list.IssuedAt),
		testEntry("math", "trigonometry", score.Competent, list.IssuedAt),
		testEntry("pedagogy", "lesson-planning", score.Competent, list.IssuedAt),
		testEntry("pedagogy", "classroom-management", score.Fluent, list.IssuedAt),
	}

	// Act
	result, err := list.Interpret(entries, "operators")

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Passed)
	assert.True(t, result.Requirements["all-fundamentals"].Passed)
	assert.True(t, result.Requirements["any-advanced"].Passed)
	assert.True(t, result.Requirements["at-least-2-breadth"].Passed)
}

func TestListInterpretDoesNotFilterExpiredEntries(t *testing.T) {
	// Description
	// Interpretation follows the result contract and does not invent expiry filtering.

	// Arrange
	list := validInterpretationList()
	entry := testEntry("math", "addition", score.Competent, list.IssuedAt)
	entry.ValidUntil = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)

	// Act
	result, err := list.Interpret([]transcript.Entry{entry}, "arithmetic")

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Requirements["math.addition"].Passed)
}

func TestListInterpretSelectsHighestDuplicateScore(t *testing.T) {
	// Description
	// Duplicate entries resolve to the highest score before timestamp tie-breakers.

	// Arrange
	list := validInterpretationList()
	olderHigh := testEntry("math", "addition", score.Competent, list.IssuedAt.Add(-time.Hour))
	newerLow := testEntry("math", "addition", score.Aware, list.IssuedAt.Add(time.Hour))

	// Act
	result, err := list.Interpret([]transcript.Entry{newerLow, olderHigh}, "arithmetic")

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Requirements["math.addition"].Passed)
}

func TestSelectEntriesUsesLatestIssuedAtThenValidUntil(t *testing.T) {
	// Description
	// Equal duplicate scores resolve by issued-at and then valid-until deterministically.

	// Arrange
	base := time.Date(2026, time.January, 26, 1, 0, 0, 0, time.UTC)
	old := testEntry("math", "addition", score.Competent, base)
	newer := testEntry("math", "addition", score.Competent, base.Add(time.Hour))
	newerShort := newer
	newerShort.ValidUntil = base.Add(24 * time.Hour)
	newerLong := newer
	newerLong.ValidUntil = base.Add(48 * time.Hour)

	// Act
	selected := selectEntries([]transcript.Entry{newerShort, old, newerLong})

	// Assert
	chosen := selected[entryKey{
		owner:   "example.com",
		name:    "math",
		version: "0.1.0",
		topic:   "addition",
	}]
	assert.Equal(t, newerLong.IssuedAt, chosen.IssuedAt)
	assert.Equal(t, newerLong.ValidUntil, chosen.ValidUntil)
}

func TestListInterpretRejectsUnknownInterpretation(t *testing.T) {
	// Description
	// Looking up an undeclared interpretation returns a descriptive error.

	// Arrange
	list := validInterpretationList()

	// Act
	result, err := list.Interpret(nil, "missing")

	// Assert
	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), `unknown interpretation "missing"`)
}

func testEntry(listName, topicID string, value score.Score, issuedAt time.Time) transcript.Entry {
	return transcript.Entry{
		UserEmail:        "first.last@example.com",
		TopicListOwner:   "example.com",
		TopicList:        listName,
		TopicListVersion: "0.1.0",
		Topic:            topicID,
		Score:            value,
		IssuedAt:         issuedAt,
		ValidUntil:       issuedAt.AddDate(2, 0, 0),
		IssuedBy:         "example.com",
	}
}
