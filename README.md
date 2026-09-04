# opm-go

A Go library for the [Open Proficiency Model](https://github.com/openproficiency/model) (OPM).

`opm-go` provides native Go types and operations for the full lifecycle of proficiency data

- defining knowledge domains
- scoring proficiency
- issuing signed transcript entries
- verifying authenticity
- and interpreting scores into badges, levels, and roles.

## Features

- **Native types** for every OPM entity — `Topic`, `TopicList`, `TranscriptEntry`, `Transcript`,
  `ScoreInterpretation`, and `ScoreInterpretationList`.
- **Encoding** to and from JSON and YAML that matches the `@openproficiency/schema` definitions exactly.
- **Schema validation** against the bundled, versioned OPM JSON Schemas.
- **Signing and verification** using detached OpenPGP (GPG) signatures over the protected fields of
  topic lists, transcript entries, and score interpretation lists.
- **Scores** as the canonical named levels: `unaware`, `aware`, `familiar`, `competent`, `fluent`.
- **Interpretation** of a set of scores against requirement expressions (`all`, `any`, `at-least-N`) to
  award badges, proficiency levels, or job roles.
- **Dependency resolution** for topic lists that reference other issuers' lists.

## Install

Install latest version

```bash
go get github.com/openproficiency/opm-go@latest
```

View Available Versions

```
go list -m -versions github.com/openproficiency/opm-go
```

Requires Go 1.23 or later.

## Get Started

> **Note:** Signatures are not shown in the below examples.
> See the full documentation for signing processes.

### Load from files

`math.yml`

```yaml
owner: example.com
name: math
description: Basic mathematics
version: 0.1.0
issued-at: 2026-09-01T00:00:00Z
signature: null
signed-by: null
topics:
  addition:
    description: Combining quantities
  subtraction:
    description: Finding the difference between quantities
  multiplication:
    description: Repeated addition of a quantity
  division:
    description: Splitting a quantity into equal parts
```

`math-levels.yml`

```yaml
owner: example.com
name: math-levels
description: Mathematics proficiency levels
version: 0.1.0
issued-at: 2026-09-01T00:00:00Z
signature: null
signed-by: null
score-interpretations:
  arithmetic-1:
    name: Arithmetic - Level 1
    description: Competent in basic arithmetic operations
    requirements:
      math.addition: competent
      math.subtraction: competent
      math.multiplication: competent
      math.division: competent
dependencies:
  math: example.com/math@0.1.0
```

`transcript-entries.yml`

```yaml
- user-email: learner@example.com
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
- user-email: learner@example.com
  topic-list-owner: example.com
  topic-list: math
  topic-list-version: 0.1.0
  topic: multiplication
  score: competent
  issued-at: 2026-09-01T00:00:00Z
  valid-until: 2028-09-01T00:00:00Z
  issued-by: example.com
- user-email: learner@example.com
  topic-list-owner: example.com
  topic-list: math
  topic-list-version: 0.1.0
  topic: division
  score: competent
  issued-at: 2026-09-01T00:00:00Z
  valid-until: 2028-09-01T00:00:00Z
  issued-by: example.com
```

```go
package main

import (
	"fmt"
	"os"

	"github.com/openproficiency/opm-go/scoreinterpretation"
	"github.com/openproficiency/opm-go/topic"
	"github.com/openproficiency/opm-go/transcript"
)

func main() {
	// 1. Load a topic list.
	mathTopicList, _ := os.ReadFile("math.yml")
	var math topic.List
	_ = math.UnmarshalYAML(mathTopicList)

	// 2. Load an interpretation list.
	levelsInterpretationList, _ := os.ReadFile("math-levels.yml")
	var levels scoreinterpretation.List
	_ = levels.UnmarshalYAML(levelsInterpretationList)

	// 3. Load transcript entries.
	userTranscriptEntries, _ := os.ReadFile("transcript-entries.yml")
	var entries transcript.Entries
	_ = entries.UnmarshalYAML(userTranscriptEntries)

	// 4. Compare the transcript with the interpretation.
	result, _ := levels.Interpret(entries, "arithmetic-1")

	// 5. View the results.
	fmt.Println(result.Passed)                                     // true
	fmt.Println(result.Requirements["math.addition"].Passed)       // true
	fmt.Println(result.Requirements["math.subtraction"].Passed)    // true
	fmt.Println(result.Requirements["math.multiplication"].Passed) // true
	fmt.Println(result.Requirements["math.division"].Passed)       // true
}
```

### Fully in Code

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
	issuedAt := time.Now()

	// 1. Define a knowledge space
	math := topic.List{
		Owner:       "example.com",
		Name:        "math",
		Description: "Basic mathematics",
		Version:     "0.1.0",
		IssuedAt:    issuedAt,
		Topics: map[string]topic.Topic{
			"addition":       {ID: "addition", Description: "Combining quantities"},
			"subtraction":    {ID: "subtraction", Description: "Finding a difference"},
			"multiplication": {ID: "multiplication", Description: "Repeated addition"},
			"division":       {ID: "division", Description: "Splitting into equal parts"},
		},
	}

	// 2. Define an interpretation
	levels := scoreinterpretation.List{
		Owner:       "example.com",
		Name:        "math-levels",
		Description: "Mathematics proficiency levels",
		Version:     "0.1.0",
		IssuedAt:    issuedAt,
		Dependencies: map[string]topic.Dependency{
			"math": {Owner: math.Owner, Name: math.Name, Version: math.Version},
		},
		Interpretations: map[string]scoreinterpretation.Interpretation{
			"arithmetic-1": {
				ID:          "arithmetic-1",
				Name:        "Arithmetic - Level 1",
				Description: "Competent in basic arithmetic operations",
				Requirements: []scoreinterpretation.Requirement{
					scoreinterpretation.Require("math", "addition", score.Competent),
					scoreinterpretation.Require("math", "subtraction", score.Competent),
					scoreinterpretation.Require("math", "multiplication", score.Competent),
					scoreinterpretation.Require("math", "division", score.Competent),
				},
			},
		},
	}

	// 3. Issue scores
	entry := func(topicID string) transcript.Entry {
		return transcript.Entry{
			UserEmail:        "learner@example.com",
			TopicListOwner:   math.Owner,
			TopicList:        math.Name,
			TopicListVersion: math.Version,
			Topic:            topicID,
			Score:            score.Competent,
			IssuedAt:         issuedAt,
			ValidUntil:       issuedAt.AddDate(2, 0, 0),
			IssuedBy:         "example.com",
		}
	}
	entries := []transcript.Entry{
		entry("addition"),
		entry("subtraction"),
		entry("multiplication"),
		entry("division"),
	}

	// 4. Compare the transcript with the interpretation
	result, err := levels.Interpret(entries, "arithmetic-1")
	if err != nil {
		panic(err)
	}

	// 5. View the results
	fmt.Println(result.Passed)                                     // true
	fmt.Println(result.Requirements["math.addition"].Passed)       // true
	fmt.Println(result.Requirements["math.subtraction"].Passed)    // true
	fmt.Println(result.Requirements["math.multiplication"].Passed) // true
	fmt.Println(result.Requirements["math.division"].Passed)       // true
}
```

## Packages

Full documentation lives in the [`docs`](docs/) directory.

| Package                                               | Purpose                                                            |
| ----------------------------------------------------- | ------------------------------------------------------------------ |
| [`topic`](docs/topic.md)                              | Signed topic lists, topic relationships, and dependency handling.  |
| [`score`](docs/score.md)                              | Ordered OPM score values and string conversion.                    |
| [`transcript`](docs/transcript.md)                    | Unsigned storage and signed transcript exports.                    |
| [`scoreinterpretation`](docs/score-interpretation.md) | Score interpretation requirements, lists, evaluation, and signing. |
