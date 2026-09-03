# Topic

The `topic` package provides the structure to define a knowledge area.

It is based on the Open Proficiency [Topic List Interpretation](https://github.com/openproficiency/model/blob/main/specs/topic-list.md) specification.

## Topic list

```go
package main

import (
	"fmt"

	"github.com/openproficiency/opm-go/topic"
)

func main() {
	topicList := topic.List{
		Owner:       "example.com",
		Name:        "math",
		Description: "Math topics through basic calculus",
		Version:     "0.1.0",

		// Optionally declare topics inline. Map keys and topic IDs must match.
		Topics: map[string]topic.Topic{
			"arithmetic": {
				ID:          "arithmetic",
				DisplayName: "Arithmetic",
				Description: "Basic operations for numeric calculations",
				Subtopics:   []string{"addition", "subtraction", "multiplication", "division"},
			},
			"algebra": {
				ID:          "algebra",
				Description: "Working with variables and equations",
				Subtopics:   []string{"variables", "constants", "single-variable-equations"},
				Pretopics:   []string{"arithmetic"},
			},
		},
	}

	// Optionally add topics one at a time.
	topicList.Add(topic.Topic{
		ID: "calculus",
		// DisplayName: "Calculus", // Optional
		Description: "Rates of change and accumulation",
		Pretopics:   []string{"algebra"},
	})

  // View topic list details
	fmt.Println(topicList.Owner)       // example.com
	fmt.Println(topicList.Name)        // math
	fmt.Println(len(topicList.Topics)) // 2
  fmt.Println(topicList.Topics["arithmetic"].DisplayName) // Arithmetic
}
```

## Convenience Methods

The `Topic.List` provides a few methods for convenience.

### `List.FullName()`

This provides the fully qualified topic list name like `example.com/math@0.1.0`

```go
fmt.Println(topicList.FullName()) // example.com/math@0.1.0
```

### `List.AtomicTopics()`

- Returns topics without subtopics.
- Returns a new `map[string]topic.Topic` keyed by topic ID.

```go
atomicTopics := topicList.AtomicTopics()

fmt.Println(len(atomicTopics))       // 1
fmt.Println(atomicTopics["calculus"].ID) // calculus
fmt.PrintLn(len(groupTopics["calculus"])) // 0
```

### `List.GroupTopics()`

- Returns topics with subtopics.
- Returns a new `map[string]topic.Topic` keyed by topic ID.

```go
groupTopics := topicList.GroupTopics()

fmt.Println(len(groupTopics))          // 2
fmt.Println(groupTopics["arithmetic"].ID) // arithmetic
fmt.PrintLn(len(groupTopics["arithmetic"])) // 4
```

### `ComplexityReport(List)`

- Returns basic list-scope metrics.
- Helps identify large or deeply grouped topic lists.

| Field              | Description                             |
| ------------------ | --------------------------------------- |
| `TopicCount`       | Total topics.                           |
| `AtomicTopicCount` | Topics without subtopics.               |
| `GroupTopicCount`  | Topics with subtopics.                  |
| `MaxSubtopics`     | Largest direct subtopic count.          |
| `MaxDepth`         | Deepest local subtopic hierarchy level. |

```go
report := List.ComplexityReport(topicList)

fmt.Println(report.TopicCount)         // 3
fmt.Println(report.AtomicTopicCount)   // 1
fmt.Println(report.GroupTopicCount)    // 2
fmt.Println(report.MaxSubtopics)       // 4
fmt.Println(report.MaxDepth)           // 2
fmt.Println(report.DependenciesCount)  // 0
```

## Encode / Decode

The Topic `List` provides methods to import and exports in both YAML and JSON,
per Open Proficiency Model [Topic List Schema](https://github.com/openproficiency/model/blob/main/schemas/topic-list.schema.json).

### YAML

- `MarshalYAML() ([]byte, error)` returns OPM-compatible YAML.
- `UnmarshalYAML(data []byte) error` loads OPM-compatible YAML.

```go
package main

import (
	"fmt"
	"time"

	"github.com/openproficiency/opm-go/topic"
)

func main() {
	topicList := topic.List{
		Owner:       "example.com",
		Name:        "math",
		Description: "Math topics through basic calculus",
		Version:     "0.1.0",
		IssuedAt:    time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		Topics: map[string]topic.Topic{
			"addition": {
				ID:          "addition",
				DisplayName: "Addition",
				Description: "Combining quantities into a total",
			},
			"subtraction": {
				ID:          "subtraction",
				DisplayName: "Subtraction",
				Description: "Finding the difference between quantities",
			},
		},
	}

	// Export YAML
	yamlData, _ := topicList.MarshalYAML()
	fmt.Println(string(yamlData))

	// Import YAML
	var importedTopicList topic.List
	_ = importedTopicList.UnmarshalYAML(yamlData)

  // View topic list details
	fmt.Println(importedTopicList.Owner) // example.com
	fmt.Println(importedTopicList.Name) // math
	fmt.Println(len(importedTopicList.Topics)) // 2
	fmt.Println(importedTopicList.Topics["addition"].DisplayName) // Addition
}
```

### JSON

- `MarshalJSON() ([]byte, error)` returns OPM-compatible JSON.
- `UnmarshalJSON(data []byte) error` loads OPM-compatible JSON.

```go
package main

import (
	"fmt"
	"time"

	"github.com/openproficiency/opm-go/topic"
)

func main() {
	topicList := topic.List{
		Owner:       "example.com",
		Name:        "math",
		Description: "Math topics through basic calculus",
		Version:     "0.1.0",
		IssuedAt:    time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		Topics: map[string]topic.Topic{
			// Arithmetic operations
			"addition": {
				ID:          "addition",
				DisplayName: "Addition",
				Description: "Combining quantities into a total",
			},
			"subtraction": {
				ID:          "subtraction",
				DisplayName: "Subtraction",
				Description: "Finding the difference between quantities",
			},
		},
	}

	// Export JSON
	jsonData, _ := topicList.MarshalJSON()
	fmt.Println(string(jsonData))

	// Import JSON
	var importedTopicList topic.List
	_ = importedTopicList.UnmarshalJSON(jsonData)

  // View topic list details
	fmt.Println(importedTopicList.Owner) // example.com
	fmt.Println(importedTopicList.Name) // math
	fmt.Println(len(importedTopicList.Topics)) // 2
	fmt.Println(importedTopicList.Topics["addition"].DisplayName) // Addition
}
```

## Signing

To distribute a Topic List, it must have be signed with a GPG key.

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
)

func main() {

	// Create a topic list
	topicList := topic.List{
		Owner:       "example.com",
		Name:        "math",
		Description: "Math topics through basic calculus",
		Version:     "0.1.0",
		IssuedAt:    time.Now(),
		Topics: map[string]topic.Topic{
			"arithmetic": {
				ID:          "arithmetic",
				Description: "Basic operations for numeric calculations",
			},
		},
	}

	// Load private Key
	privateKeys, _ := openpgp.ReadArmoredKeyRing(strings.NewReader(os.Getenv("OPM_PRIVATE_KEY")))
	privateKey := privateKeys[0]

	// Sign topic list
	topicList.Sign(privateKey, os.Getenv("OPM_KEY_PASSPHRASE"))

  // Show signing details
	signedBy, _ := topicList.SignedBy()
	signature, _ := topicList.Signature()
	fmt.Printf("SignedBy: %s\n", signedBy)
	fmt.Printf("Signature: %s\n", signature)
	keyID, _ := topicList.SignatureKeyID()
	fmt.Printf("Signature key ID: %016X\n", keyID)

	// Load public key
	publicKeys, _ := openpgp.ReadArmoredKeyRing(strings.NewReader(os.Getenv("OPM_PUBLIC_KEY")))
	publicKey := publicKeys[0]

	// Verify the list's signature against the public key
	valid, _ := topicList.Verify(publicKey)
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

## Syntax Validation

The `List.Validate()` provides a basic syntax verification.

## Dependency Validation

Dependency validation is not currently in scope.

- Topic lists are assumed to exist at the specified location.
- Topic lists are assumed to contain the required topics.
