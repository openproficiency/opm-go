# Score interpretations

The `scoreinterpretation` package provides the structure to define official ways of interpretting a user's transcript entries.

It is based on the Open Proficiency [Topic Score Interpretation](https://github.com/openproficiency/model/blob/main/specs/score-interpretation-list.md) specification.

```go
import ( "github.com/openproficiency/opm-go/scoreinterpretation" )
```

## Score Interpretation

| Field          | Required | Description                                  |
| -------------- | -------- | -------------------------------------------- |
| `ID`           | Yes      | Kebab-case identifier.                       |
| `Name`         | Yes      | Display name.                                |
| `Description`  | Yes      | Description of the outcome.                  |
| `Requirements` | Yes      | Required topic scores and logical operators. |

## Requirement operators

`Requirements` represents one topic score requirement or a logical combination of requirements.

| API                                                      | Result                                                     |
| -------------------------------------------------------- | ---------------------------------------------------------- |
| `Require(dependency, topic string, minimum score.Score)` | One topic and its minimum score.                           |
| `All{ID, Requirements}`                                  | All nested requirements must be satisfied.                 |
| `Any{ID, Requirements}`                                  | At least one nested requirement must be satisfied.         |
| `AtLeast{ID, MinCount, Requirements}`                    | At least `MinCount` nested requirements must be satisfied. |

- An `ID` must be unique within the that layer of requirements group

### `all`

Every nested requirement must pass.

- An `All` at top-level is implied and hence not necessary.

```go
package main

import (
	"fmt"

	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/scoreinterpretation"
)

func main() {

	interpretation := scoreinterpretation.Interpretation{
		ID:          "arithmetic-1",
		Name:        "Arithmetic - Level 1",
		Description: "Practical experience with arithmetic operations.",
		Requirements: []scoreinterpretation.Requirement{
			// Implicit top-level all
			scoreinterpretation.Require("math", "addition", score.Competent),
			scoreinterpretation.Require("math", "subtraction", score.Competent),
			// Explicit all - for demonstration. Not a good example
			scoreinterpretation.All{
				ID: "advanced-arithmetic",
				Requirements: []scoreinterpretation.Requirement{
					scoreinterpretation.Require("math", "multiplication", score.Competent),
					scoreinterpretation.Require("math", "division", score.Competent),
				},
			},
		},
	}

	// Show interpretation details
	fmt.Println(interpretation.ID) // arithmetic-1
	fmt.Println(interpretation.Name) // Arithmetic - Level 1
	fmt.Println(len(interpretation.Requirements)) // 3

}
```

### `any`

At least one nested requirement must pass.

```go
package main

import (
	"fmt"

	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/scoreinterpretation"
)

func main() {

	interpretation := scoreinterpretation.Interpretation{
		ID:          "math-tutor-1",
		Name:        "Math Tutor Level 1",
		Description: "Qualified to tutor using arithmetic, advanced math, and classroom management.",
		Requirements: []scoreinterpretation.Requirement{
			// All arithmetic operations
			scoreinterpretation.Require("math", "addition", score.Competent),
			scoreinterpretation.Require("math", "subtraction", score.Competent),
			scoreinterpretation.Require("math", "multiplication", score.Competent),
			scoreinterpretation.Require("math", "division", score.Competent),
			// Any advanced subject
			scoreinterpretation.Any{
				ID: "any-advanced-subject",
				Requirements: []scoreinterpretation.Requirement{
					scoreinterpretation.Require("math", "trigonometry", score.Competent),
					scoreinterpretation.Require("math", "calculus", score.Competent),
				},
			},
		},
	}

	// Show interpretation details
	fmt.Println(interpretation.ID) // math-tutor-1
	fmt.Println(interpretation.Name) // Math Tutor Level 1
	fmt.Println(len(interpretation.Requirements)) // 5

}
```

### `at-least-N`

At least `MinCount` nested requirements must pass.

```go
package main

import (
	"fmt"

	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/scoreinterpretation"
)

func main() {

	interpretation := scoreinterpretation.Interpretation{
		ID:          "arithmetic-breadth",
		Name:        "Arithmetic Breadth",
		Description: "Competent in at least two arithmetic operations.",
		Requirements: []scoreinterpretation.Requirement{
			// At least 2 arithmetic operations
			scoreinterpretation.AtLeast{
				ID: "at-least-2-arithmetic-operations",
				MinCount: 2,
				Requirements: []scoreinterpretation.Requirement{
					scoreinterpretation.Require("math", "addition", score.Competent),
					scoreinterpretation.Require("math", "subtraction", score.Competent),
					scoreinterpretation.Require("math", "multiplication", score.Competent),
					scoreinterpretation.Require("math", "division", score.Competent),
				},

			},
		},
	}

	// Show details
	fmt.Println(interpretation.ID) // arithmetic-breadth
	fmt.Println(interpretation.Name) // Arithmetic Breadth
	fmt.Println(len(interpretation.Requirements)) // 1

}
```

### Nested requirements

Requirements may be nested to build requirements groups.

```go
package main

import (
	"fmt"

	"github.com/openproficiency/opm-go/score"
	si "github.com/openproficiency/opm-go/scoreinterpretation"
)

func main() {

	interpretation := si.Interpretation{
		ID:          "math-tutor-1",
		Name:        "Math Tutor Level 1",
		Description: "Qualified to tutor using arithmetic, advanced math, and pedagogy skills.",
		Requirements: []si.Requirement{

			// All arithmetic operations
			si.Require("math", "addition", score.Competent),
			si.Require("math", "subtraction", score.Competent),
			si.Require("math", "multiplication", score.Competent),
			si.Require("math", "division", score.Competent),

			// Classroom management
			si.Require("std-pedagogy", "classroom-management", score.Competent),

			// Any advanced branch
			si.Any{
				ID: "any-advanced-subject",
				Requirements: []si.Requirement{
					si.Require("math", "trigonometry", score.Competent),
					si.Require("math", "calculus", score.Competent),
				},
			},

			// At least 2 pedagogy skills
			si.AtLeast{
				ID: "at-least-2-pedagogy-skills",
				MinCount: 2,
				Requirements: []si.Requirement{
					si.Require("std-pedagogy", "lesson-planning", score.Competent),
					si.Require("std-pedagogy", "lesson-customization", score.Competent),
					si.Require("std-pedagogy", "teacher-mentoring", score.Competent),
				},
			},
		},
	}

	// Show details
	fmt.Println(interpretation.ID) // math-tutor-1
	fmt.Println(interpretation.Name) // Math Tutor Level 1
	fmt.Println(len(interpretation.Requirements)) // 7

}
```

# Score Interpretation List

A `List` groups related interpretations into a bumdle to enable distribution.

| Field             | Required | Description                                             |
| ----------------- | -------- | ------------------------------------------------------- |
| `Owner`           | Yes      | Domain name of the entity maintaining the list.         |
| `Name`            | Yes      | Kebab-case list name.                                   |
| `Description`     | Yes      | Description of the list.                                |
| `Version`         | Yes      | Semantic version.                                       |
| `IssuedAt`        | Yes      | Time this version was created.                          |
| `Interpretations` | Yes      | Map of identifiers to interpretations.                  |
| `Dependencies`    | No       | Map of kebab-case aliases to `topic.Dependency` values. |

```go
package main

import (
	"fmt"
	"time"

	"github.com/openproficiency/opm-go/topic"
	"github.com/openproficiency/opm-go/score"
	si "github.com/openproficiency/opm-go/scoreinterpretation"
)

func main() {
	// Existing score interpretation list
	interpretationList := si.List{
		Owner:       "example.com",
		Name:        "math-teacher-levels",
		Description: "Internal definition of math teacher proficiency",
		Version:     "0.1.0",
		IssuedAt:    time.Now(),
		Dependencies: map[string]topic.Dependency{
			"std-math": {
				Owner:     "example.com",
				Name:      "math",
				Version:   "0.1.0",
				Locations: []string{
					"https://example.com/0.1.0/math.yml",
					"npm:@example/math-topics@0.1.0",
				},
			},
			"std-pedagogy": {
				Owner:     "example.com",
				Name:      "pedagogy",
				Version:   "0.1.0",
				Locations: []string{
					"https://example.com/0.1.0/pedagogy.yml",
					"npm:@example/pedagogy-topics@0.1.0",
				},
			},
		},
		Interpretations: map[string]si.Interpretation{
			"math-teacher-junior": {
				ID:          "math-teacher-junior",
				Name:        "JR. Math Teacher",
				Description: "Able to support another teacher.",
				Requirements: []si.Requirement{
					// Math topics
					si.Require("std-math", "addition", score.Competent),
					si.Require("std-math", "subtraction", score.Competent)
					// Pedagogy topics
					si.Require("std-pedagogy", "classroom-management", score.Competent),
				},
			},
		},
	}

	// Optionally add score interpretations one at a time
	interpretationList.Add(si.Interpretation{
		ID:          "math-teacher",
		Name:        "Math Teacher",
		Description: "Able to teach a classroom alone.",
		Requirements: []si.Requirement{
			// Math topics
			si.Require("std-math", "multiplication", score.Competent),
			si.Require("std-math", "division", score.Competent),
			// Pedagogy topics
			si.Require("std-pedagogy", "lesson-planning", score.Competent),
			si.Require("std-pedagogy", "lesson-customization", score.Competent),
		},
	})

}
```

## Encode / Decode

The Score Interpretation `List` provides methods to import and exports in both YAML and JSON,
per Open Proficiency Model [Score Intepretation Schema](https://github.com/openproficiency/model/blob/main/schemas/score-interpretation-list.schema.json).

### YAML

- `MarshalYAML() ([]byte, error)` - Returns OPM schema compliant YAML.
- `UnmarshalYAML(data []byte) error` - Loads OPM schema compliant YAML.

```go
package main

import (
	"fmt"
	"time"
	"github.com/openproficiency/opm-go/topic"
	"github.com/openproficiency/opm-go/score"
	si "github.com/openproficiency/opm-go/scoreinterpretation"
)

func main() {
	// Existing score interpretation list
	interpretationList := si.List{
		Owner:       "example.com",
		Name:        "math-levels",
		Description: "Math proficiency levels",
		Version:     "0.1.0",
		IssuedAt:    time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		Dependencies: map[string]topic.Dependency{
			"math": {
				Owner:     "example.com",
				Name:      "math",
				Version:   "0.1.0",
				Locations: []string{
					"https://example.com/0.1.0/math.yml",
					"npm:@example/math-topics@0.1.0",
				},
			},
		},
		Interpretations: map[string]si.Interpretation{
			"arithmetic-1": {
				ID:          "arithmetic-1",
				Name:        "Arithmetic - Level 1",
				Description: "Practical experience with addition and subtraction.",
				Requirements: []si.Requirement{
					si.Require("math", "addition", score.Competent),
					si.Require("math", "subtraction", score.Competent),
				},
			},
		},
	}

	// Export to OPM schema YAML
	yamlData, _ := interpretationList.MarshalYAML()
	fmt.Println(string(yamlData))

	// Import from OPM schema YAML
	var importedInterpretationList si.List
	_ = importedInterpretationList.UnmarshalYAML(yamlData)

	// Show Result
	fmt.PrintLn(len(importedInterpretationList.Name)) // math-levels
	fmt.PrintLn(len(importedInterpretationList.Intepretations)) // 1
	fmt.Println(importedInterpretationList.Interpretations["arithmetic-1"].Name) // Arithmetic - Level 1
}
```

#### Development Notes

- YAML dependency: internal `gopkg.in/yaml.v3`.
- Callers do not import a YAML package.

### JSON

- `MarshalJSON() ([]byte, error)` - Returns OPM schema compliant JSON.
- `UnmarshalJSON(data []byte) error` - Loads OPM schema compliant JSON.

```go
package main

import (
	"fmt"
	"time"

	"github.com/openproficiency/opm-go/topic"
	"github.com/openproficiency/opm-go/score"
	si "github.com/openproficiency/opm-go/scoreinterpretation"
)

func main() {
	// Existing score interpretation list
	interpretationList := si.List{
		Owner:       "example.com",
		Name:        "math-levels",
		Description: "Math proficiency levels",
		Version:     "0.1.0",
		IssuedAt:    time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		Dependencies: map[string]topic.Dependency{
			"math": {
				Owner:     "example.com",
				Name:      "math",
				Version:   "0.1.0",
				Locations: []string{
					"https://example.com/0.1.0/math.yml",
					"npm:@example/math-topics@0.1.0",
				},
			},
		},
		Interpretations: map[string]si.Interpretation{
			"arithmetic-1": {
				ID:          "arithmetic-1",
				Name:        "Arithmetic - Level 1",
				Description: "Practical experience with addition and subtraction.",
				Requirements: []si.Requirement{
					si.Require("math", "addition", score.Competent),
					si.Require("math", "subtraction", score.Competent),
				},
			},
		},
	}

	// Export the list to OPM schema JSON.
	jsonData, _ := interpretationList.MarshalJSON()
	fmt.Println(string(jsonData))

	// Import OPM JSON into a new list.
	var importedInterpretationList si.List
	_ = importedInterpretationList.UnmarshalJSON(jsonData)

	// Show Result
	fmt.PrintLn(len(importedInterpretationList.Name)) // math-levels
	fmt.PrintLn(len(importedInterpretationList.Intepretations)) // 1
	fmt.Println(importedInterpretationList.Interpretations["arithmetic-1"].Name) // Arithmetic - Level 1
}
```

## Signing

To distribute a Score Interpretation List, it must have be signed with a GPG key.

- `Sign(privateKey *openpgp.Entity, passphrase string) error` - Use a private key to sign the list.
- `Signature() (string, error)` - Return the signature.
- `SignedBy() (string, error)` - Return the email used for signing.
- `SignatureKeyID() (uint64, error)` - Return the Key ID embedded in the signature.
- `Verify(publicKey *openpgp.Entity) (bool, error)` - Report whether protected content matches the
  signature and the issuer matches the public key.

```go
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/openproficiency/opm-go/topic"
	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/scoreinterpretation"
)

func main() {
	// Existing score interpretation list
	interpretationList := scoreinterpretation.List{
		Owner:       "example.com",
		Name:        "math-levels",
		Description: "Math proficiency levels",
		Version:     "0.1.0",
		IssuedAt:    time.Now(),
		Dependencies: map[string]topic.Dependency{
			"math": {
				Owner:     "example.com",
				Name:      "math",
				Version:   "0.1.0",
				Locations: []string{
					"https://example.com/0.1.0/math.yml",
					"npm:@example/math-topics@0.1.0",
				},
			},
		},
		Interpretations: map[string]scoreinterpretation.Interpretation{
			"arithmetic-1": {
				ID:          "arithmetic-1",
				Name:        "Arithmetic - Level 1",
				Description: "Practical experience with addition and subtraction.",
				Requirements: []scoreinterpretation.Requirement{
					scoreinterpretation.Require("math", "addition", score.Competent),
					scoreinterpretation.Require("math", "subtraction", score.Competent),
				},
			},
		},
	}

	// Load the private key
	privateKeys, _ := openpgp.ReadArmoredKeyRing(strings.NewReader(os.Getenv("OPM_PRIVATE_KEY")))
	privateKey := privateKeys[0]

	// Sign the topic list
	interpretationList.Sign(privateKey, os.Getenv("OPM_KEY_PASSPHRASE"))

	// Show signing details
	signedBy, _ := interpretationList.SignedBy()
	keyID, _ := interpretationList.SignatureKeyID()
	signature, _ := interpretationList.Signature()
	fmt.Printf("SignedBy: %s\n", signedBy)
	fmt.Printf("Signature key ID: %016X\n", keyID)
	fmt.Printf("Signature: %s\n", signature)

	// Load the public key
	publicKeys, _ := openpgp.ReadArmoredKeyRing(strings.NewReader(os.Getenv("OPM_PUBLIC_KEY")))
	publicKey := publicKeys[0]

	// Verify the list's signature against the public key
	valid, _ := interpretationList.Verify(publicKey)
	fmt.Printf("Signature valid: %t\n", valid)
}
```

### Signature Staleness

Any changes to protected fields, cause the stored signature to becom stale.
In this case, related methos will return empty results and error messages.

- `Signature()` returns `"", ErrSignatureStale`.
- `SignedBy()` returns `"", ErrSignatureStale`.
- `SignatureKeyID()` returns `0, ErrSignatureStale`.
- `Verify()` returns `false, ErrSignatureStale`.
- JSON encoding succeeds with `signature` set to `null` and `signed-by` set to `null`.
- YAML encoding succeeds with `signature` set to `null` and `signed-by` set to `null`.

#### Development Notes

- `List.Sign` stores a canonical protected-field hash.
- Signature methods compare current fields with that hash.
- Signature metadata is private.
- Calling `Sign` again replaces the old signature.
- A list that has never been signed encodes `signature` and `signed-by` as `null`.

## Syntax Validation

The `Interpretation.Validate()` provides a basic syntax verification.

#### Invalid interpretation ID

```go
interpretation.ID = "Math Tutor"
err := interpretation.Validate()
```

#### Missing operator ID

```go
interpretation.Requirements = []scoreinterpretation.Requirement{
	scoreinterpretation.Any{
		// ID: "any-advanced-subject"
		Requirements: []scoreinterpretation.Requirement{
			scoreinterpretation.Require("math", "trigonometry", score.Competent),
			scoreinterpretation.Require("math", "calculus", score.Competent),
		},
	},
}
err := interpretation.Validate()
```

#### Duplicate operator IDs

```go
interpretation.Requirements = []scoreinterpretation.Requirement{
	scoreinterpretation.Any{
		ID: "any-pathway",
		Requirements: []scoreinterpretation.Requirement{
			scoreinterpretation.Require("math", "trigonometry", score.Competent),
		},
	},
	scoreinterpretation.All{
		ID: "any-pathway",
		Requirements: []scoreinterpretation.Requirement{
			scoreinterpretation.Require("math", "addition", score.Competent),
		},
	},
}
err := interpretation.Validate()
```

#### Duplicate qualified topic

```go
interpretationList.Interpretations = map[string]scoreinterpretation.Interpretation{
	"arithmetic-1": {
		ID: "arithmetic-1",
		Requirements: []scoreinterpretation.Requirement{
			scoreinterpretation.Require("math", "addition", score.Competent),
		},
	},
	"arithmetic-2": {
		ID: "arithmetic-2",
		Requirements: []scoreinterpretation.Requirement{
			scoreinterpretation.Require("math", "addition", score.Fluent),
		},
	},
}
err := interpretationList.Validate()
```

### Development Notes

- `Interpretation.Validate() error` validates fields and the complete requirement tree.
- `List.Validate() error` validates list metadata, dependencies, map keys, interpretations, and
  requirements.
- Encoding and signing validate before producing output.

- Schema constraints include hostnames, kebab-case IDs, semantic versions, URIs, topics, scores, and
  operators.
- Every dependency alias must identify a declared dependency.
- Operator IDs must be unique within an interpretation.
- Qualified topic references must be unique across a list's interpretations.
- `AtLeast.MinCount` must be positive.
- `MinCount` may exceed the requirement count, but the operator can never pass.

## Dependency Validation

Dependency validation is not currently in scope.

- Topic lists are assumed to exist at the specified location.
- Topic lists are assumed to contain the required topics.
