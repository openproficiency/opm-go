# Transcript Entry

## Score Options

`score.Score` is ordered from `Unaware` through `Fluent`. Its `String` method returns the lowercase
value required by the OPM schema.

```go
package main

import (
	"fmt"

	"github.com/openproficiency/opm-go/score"
)

func main() {
	// Score options
	fmt.Println(score.Unaware.String())  // unaware
	fmt.Println(score.Aware.String())    // aware
	fmt.Println(score.Familiar.String()) // familiar
	fmt.Println(score.Competent.String()) // competent
	fmt.Println(score.Fluent.String())   // fluent

	// Comparing scores
	fmt.Println(score.Unaware < score.Aware)       // true
	fmt.Println(score.Aware < score.Familiar)      // true
	fmt.Println(score.Familiar < score.Competent)  // true
	fmt.Println(score.Competent < score.Fluent)    // true
}
```

## Create a transcript entry (not signed)

```go
package main

import (
	"fmt"
	"time"

	"github.com/openproficiency/opm-go/topic"
	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/transcript"
)

func main() {
	// Existing topic list
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

	// Create a transcriptEntry (using existing list)
	transcriptEntry1 := transcript.Entry{
		// Source of the topic
		TopicListOwner:   topicList.Owner,
		TopicList:        topicList.Name,
		TopicListVersion: topicList.Version,
		// Associate the score with a user
		UserEmail:        "first.last@example.com",
		Topic:            topicList.Topics["arithmetic"].ID,
		Score:            score.Competent,
		IssuedAt:         time.Now(),
		ValidUntil:       time.Now().AddDate(2, 0, 0),
		// Who claims this score for the user
		IssuedBy:         "example.com",
	}
	fmt.Printf("%s - %s: %s\n", transcriptEntry1.UserEmail, transcriptEntry1.Topic, transcriptEntry1.Score)

	// Or fill out manually (theoretically)
	transcriptEntry2 := transcript.Entry{
		// Source of the topic
		TopicListOwner:   "example.com",
		TopicList:        "math",
		TopicListVersion: "0.1.0",
		// Associate the score with a user
		UserEmail:        "first.last@example.com",
		Topic:            "arithmetic",
		Score:            score.Competent,
		IssuedAt:         time.Now(),
		ValidUntil:       time.Now().AddDate(2, 0, 0),
		// Who claims this score for the user
		IssuedBy:         "example.com",
	}

	fmt.Printf("%s - %s: %s\n", transcriptEntry2.UserEmail, transcriptEntry2.Topic, transcriptEntry2.Score)
}
```

### Sign a transcript entry

```go
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/transcript"
)

func main() {

	// Existing transcript entry (not signed)
	transcriptEntry := transcript.Entry{
		UserEmail:        "first.last@example.com",
		Topic:            "arithmetic",
		TopicList:        "math",
		TopicListVersion: "0.1.0",
		TopicListOwner:   "example.com",
		Score:            score.Fluent,
		IssuedAt:         time.Now(),
		ValidUntil:       time.Now().AddDate(2, 0, 0),
		IssuedBy:         "example.com",
	}

	// Load Private key
	privateKeys, _ := openpgp.ReadArmoredKeyRing(strings.NewReader(os.Getenv("OPM_PRIVATE_KEY")))
	privateKey := privateKeys[0]

	// Sign the transcript entry
	transcriptEntry.Sign(privateKey, os.Getenv("OPM_KEY_PASSPHRASE"))

	// Load public key
	publicKeys, _ := openpgp.ReadArmoredKeyRing(strings.NewReader(os.Getenv("OPM_PUBLIC_KEY")))
	publicKey := publicKeys[0]
	valid, _ := transcriptEntry.Verify(publicKey)
	fmt.Printf("Transcript Entry valid: %t\n", valid)
}
```
