// Package transcript represents OPM transcript entries and collections.
package transcript

import (
	"fmt"
	"strings"
	"time"

	"github.com/openproficiency/opm-go/internal/canonical"
	"github.com/openproficiency/opm-go/internal/semantic"
	"github.com/openproficiency/opm-go/score"
)

// Entry is a permanent record of a user's proficiency score for one topic.
type Entry struct {
	UserEmail        string
	Topic            string
	TopicList        string
	TopicListVersion string
	TopicListOwner   string
	TopicListSources []string
	Score            score.Score
	IssuedAt         time.Time
	ValidUntil       time.Time
	IssuedBy         string
	VerificationURL  string

	schemaURL      string
	signature      string
	signedBy       string
	protectedState canonical.State
}

type protectedEntry struct {
	UserEmail        string `json:"user-email"`
	Topic            string `json:"topic"`
	TopicList        string `json:"topic-list"`
	TopicListVersion string `json:"topic-list-version"`
	TopicListOwner   string `json:"topic-list-owner"`
	Score            string `json:"score"`
	IssuedAt         string `json:"issued-at"`
	ValidUntil       string `json:"valid-until"`
	IssuedBy         string `json:"issued-by"`
	VerificationURL  string `json:"verification-url,omitempty"`
}

func (entry Entry) protected() protectedEntry {
	return protectedEntry{
		UserEmail:        entry.UserEmail,
		Topic:            entry.Topic,
		TopicList:        entry.TopicList,
		TopicListVersion: entry.TopicListVersion,
		TopicListOwner:   entry.TopicListOwner,
		Score:            entry.Score.String(),
		IssuedAt:         formatTime(entry.IssuedAt),
		ValidUntil:       formatTime(entry.ValidUntil),
		IssuedBy:         entry.IssuedBy,
		VerificationURL:  entry.VerificationURL,
	}
}

func (entry Entry) validate() error {
	if err := semantic.Email(entry.UserEmail); err != nil {
		return fmt.Errorf("validate transcript entry user email: %w", err)
	}
	if err := semantic.KebabCase(entry.Topic); err != nil {
		return fmt.Errorf("validate transcript entry topic: %w", err)
	}
	if err := semantic.KebabCase(entry.TopicList); err != nil {
		return fmt.Errorf("validate transcript entry topic list: %w", err)
	}
	if err := semantic.Semver(entry.TopicListVersion); err != nil {
		return fmt.Errorf("validate transcript entry topic list version: %w", err)
	}
	if err := semantic.Hostname(entry.TopicListOwner); err != nil {
		return fmt.Errorf("validate transcript entry topic list owner: %w", err)
	}
	for _, source := range entry.TopicListSources {
		if err := semantic.URI(source); err != nil {
			return fmt.Errorf("validate transcript entry topic list source: %w", err)
		}
	}
	if !validScore(entry.Score) {
		return fmt.Errorf("validate transcript entry score: invalid score %q", entry.Score.String())
	}
	if entry.IssuedAt.IsZero() {
		return fmt.Errorf("validate transcript entry issued at: timestamp is required")
	}
	if entry.ValidUntil.IsZero() {
		return fmt.Errorf("validate transcript entry valid until: timestamp is required")
	}
	if err := semantic.Hostname(entry.IssuedBy); err != nil {
		return fmt.Errorf("validate transcript entry issuer: %w", err)
	}
	if entry.VerificationURL != "" {
		if err := semantic.URI(entry.VerificationURL); err != nil {
			return fmt.Errorf("validate transcript entry verification URL: %w", err)
		}
		if !strings.HasPrefix(strings.ToLower(entry.VerificationURL), "https://") {
			return fmt.Errorf("validate transcript entry verification URL %q: HTTPS is required", entry.VerificationURL)
		}
	}

	return nil
}

func validScore(value score.Score) bool {
	switch value {
	case score.Unaware, score.Aware, score.Familiar, score.Competent, score.Fluent:
		return true
	default:
		return false
	}
}

func parseScore(value string) (score.Score, error) {
	switch value {
	case score.Unaware.String():
		return score.Unaware, nil
	case score.Aware.String():
		return score.Aware, nil
	case score.Familiar.String():
		return score.Familiar, nil
	case score.Competent.String():
		return score.Competent, nil
	case score.Fluent.String():
		return score.Fluent, nil
	default:
		return score.Score(0), fmt.Errorf("invalid score %q", value)
	}
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
