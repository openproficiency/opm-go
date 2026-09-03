# Score Intepretation Result

A collection of transcript entries can be compared against an interpretation to provide an `interpretation.Result`.

```go
package main

import (
	"fmt"
	"time"

	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/scoreinterpretation"
	"github.com/openproficiency/opm-go/topic"
	"github.com/openproficiency/opm-go/transcript"
)

func main() {
	issuedAt := time.Date(2026, time.January, 26, 1, 0, 0, 0, time.UTC)

	interpretationList := scoreinterpretation.List{
		Owner:       "example.com",
		Name:        "math-pathways",
		Description: "Mathematics proficiency levels with logical composition",
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
		Interpretations: map[string]scoreinterpretation.Interpretation{
			"math-tutor-1": {
				ID:          "math-tutor-1",
				Name:        "Math Tutor Level 1",
				Description: "Qualified to tutor students using arithmetic, advanced math, and pedagogy skills.",
				Requirements: []scoreinterpretation.Requirement{
					// Core subjects
					scoreinterpretation.Require("math", "addition", score.Competent),
					scoreinterpretation.Require("math", "subtraction", score.Competent),
					scoreinterpretation.Require("math", "multiplication", score.Competent),
					scoreinterpretation.Require("math", "division", score.Competent),
					scoreinterpretation.Require("std-pedagogy", "classroom-management", score.Competent),

					// Any advance subject
					scoreinterpretation.Any{
						ID: "any-advanced-subject",
						Requirements: []scoreinterpretation.Requirement{
							scoreinterpretation.Require("math", "trigonometry", score.Competent),
							scoreinterpretation.Require("math", "calculus", score.Competent),
						},
					},

					// At least 2 pedagogy skills
					scoreinterpretation.AtLeast{
						ID:       "at-least-2-pedagogy-skills",
						MinCount: 2,
						Requirements: []scoreinterpretation.Requirement{
							scoreinterpretation.Require("std-pedagogy", "lesson-planning", score.Competent),
							scoreinterpretation.Require("std-pedagogy", "lesson-customization", score.Competent),
							scoreinterpretation.Require("std-pedagogy", "teacher-mentoring", score.Competent),
						},
					},
				},
			},
		},
	}

	transcriptEntries := []transcript.Entry{
		{
			UserEmail:        "first.last@example.com",
			TopicListOwner:   "example.com",
			TopicList:        "math",
			TopicListVersion: "0.1.0",
			Topic:            "addition",
			Score:            score.Competent,
			IssuedAt:         issuedAt,
			ValidUntil:       issuedAt.AddDate(2, 0, 0),
			IssuedBy:         "example.com",
		},
		{
			UserEmail:        "first.last@example.com",
			TopicListOwner:   "example.com",
			TopicList:        "math",
			TopicListVersion: "0.1.0",
			Topic:            "subtraction",
			Score:            score.Competent,
			IssuedAt:         issuedAt,
			ValidUntil:       issuedAt.AddDate(2, 0, 0),
			IssuedBy:         "example.com",
		},
		{
			UserEmail:        "first.last@example.com",
			TopicListOwner:   "example.com",
			TopicList:        "math",
			TopicListVersion: "0.1.0",
			Topic:            "multiplication",
			Score:            score.Competent,
			IssuedAt:         issuedAt,
			ValidUntil:       issuedAt.AddDate(2, 0, 0),
			IssuedBy:         "example.com",
		},
		{
			UserEmail:        "first.last@example.com",
			TopicListOwner:   "example.com",
			TopicList:        "math",
			TopicListVersion: "0.1.0",
			Topic:            "division",
			Score:            score.Competent,
			IssuedAt:         issuedAt,
			ValidUntil:       issuedAt.AddDate(2, 0, 0),
			IssuedBy:         "example.com",
		},
		{
			UserEmail:        "first.last@example.com",
			TopicListOwner:   "example.com",
			TopicList:        "math",
			TopicListVersion: "0.1.0",
			Topic:            "trigonometry",
			Score:            score.Familiar,
			IssuedAt:         issuedAt,
			ValidUntil:       issuedAt.AddDate(2, 0, 0),
			IssuedBy:         "example.com",
		},
		{
			UserEmail:        "first.last@example.com",
			TopicListOwner:   "example.com",
			TopicList:        "pedagogy",
			TopicListVersion: "0.1.0",
			Topic:            "classroom-management",
			Score:            score.Competent,
			IssuedAt:         issuedAt,
			ValidUntil:       issuedAt.AddDate(2, 0, 0),
			IssuedBy:         "example.com",
		},
		{
			UserEmail:        "first.last@example.com",
			TopicListOwner:   "example.com",
			TopicList:        "pedagogy",
			TopicListVersion: "0.1.0",
			Topic:            "lesson-planning",
			Score:            score.Competent,
			IssuedAt:         issuedAt,
			ValidUntil:       issuedAt.AddDate(2, 0, 0),
			IssuedBy:         "example.com",
		},
	}

	// Compare transcript entries to a list's interpretation
	interpretationResult, err := interpretationList.Interpret(transcriptEntries, "math-tutor-1")
	if err != nil {
		panic(err)
	}

	// Overall result
	fmt.Println(interpretationResult.Passed) // false

	// Core Subjects
	fmt.Println(interpretationResult.Requirements["math.addition"].Passed) // true
	fmt.Println(interpretationResult.Requirements["std-pedagogy.classroom-management"].Passed) // true

	// Any Advanced Subject
	anyAdvancedSubject := interpretationResult.Requirements["any-advanced-subject"]
	fmt.Println(anyAdvancedSubject.Passed) // false
	fmt.Println(anyAdvancedSubject.Requirements["math.trigonometry"].Passed) // false
	fmt.Println(anyAdvancedSubject.Requirements["math.calculus"].Passed) // false

	// At least 2 pedagogy skills
	pedagogySkills := interpretationResult.Requirements["at-least-2-pedagogy-skills"]
	fmt.Println(pedagogySkills.Passed) // false
	fmt.Println(pedagogySkills.Requirements["std-pedagogy.lesson-planning"].Passed) // true
	fmt.Println(pedagogySkills.Requirements["std-pedagogy.lesson-customization"].Passed) // false
}
```
