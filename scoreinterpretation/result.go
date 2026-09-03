package scoreinterpretation

import (
	"fmt"

	"github.com/openproficiency/opm-go/transcript"
)

// Result reports whether an interpretation or requirement passed.
type Result struct {
	Passed       bool
	Requirements map[string]Result
}

// Interpret evaluates transcript entries against one interpretation in the list.
func (list List) Interpret(entries []transcript.Entry, interpretationID string) (Result, error) {
	if err := list.Validate(); err != nil {
		return Result{}, fmt.Errorf("interpret score interpretation list: %w", err)
	}

	interpretation, exists := list.Interpretations[interpretationID]
	if !exists {
		return Result{}, fmt.Errorf("interpret score interpretation list: unknown interpretation %q", interpretationID)
	}

	selected := selectEntries(entries)
	requirements, passed, err := list.evaluateRequirements(interpretation.Requirements, selected)
	if err != nil {
		return Result{}, fmt.Errorf("interpret %q: %w", interpretationID, err)
	}

	return Result{
		Passed:       passed,
		Requirements: requirements,
	}, nil
}

type entryKey struct {
	owner   string
	name    string
	version string
	topic   string
}

func selectEntries(entries []transcript.Entry) map[entryKey]transcript.Entry {
	selected := make(map[entryKey]transcript.Entry, len(entries))
	for _, entry := range entries {
		key := entryKey{
			owner:   entry.TopicListOwner,
			name:    entry.TopicList,
			version: entry.TopicListVersion,
			topic:   entry.Topic,
		}
		current, exists := selected[key]
		if !exists || betterEntry(entry, current) {
			selected[key] = entry
		}
	}
	return selected
}

func betterEntry(candidate, current transcript.Entry) bool {
	if candidate.Score != current.Score {
		return candidate.Score > current.Score
	}
	if !candidate.IssuedAt.Equal(current.IssuedAt) {
		return candidate.IssuedAt.After(current.IssuedAt)
	}
	return candidate.ValidUntil.After(current.ValidUntil)
}

func (list List) evaluateRequirements(
	requirements []Requirement,
	entries map[entryKey]transcript.Entry,
) (map[string]Result, bool, error) {
	results := make(map[string]Result, len(requirements))
	passed := true
	for _, requirement := range requirements {
		key, result, err := list.evaluateRequirement(requirement, entries)
		if err != nil {
			return nil, false, err
		}
		results[key] = result
		passed = passed && result.Passed
	}
	return results, passed, nil
}

func (list List) evaluateRequirement(
	requirement Requirement,
	entries map[entryKey]transcript.Entry,
) (string, Result, error) {
	switch typed := requirement.(type) {
	case requiredTopic:
		dependency := list.Dependencies[typed.dependency]
		entry, exists := entries[entryKey{
			owner:   dependency.Owner,
			name:    dependency.Name,
			version: dependency.Version,
			topic:   typed.topic,
		}]
		return typed.key(), Result{Passed: exists && entry.Score >= typed.minimum}, nil
	case All:
		return list.evaluateAll(typed.ID, typed.Requirements, entries)
	case *All:
		if typed == nil {
			return "", Result{}, fmt.Errorf("all requirement is nil")
		}
		return list.evaluateAll(typed.ID, typed.Requirements, entries)
	case Any:
		return list.evaluateAny(typed.ID, typed.Requirements, entries)
	case *Any:
		if typed == nil {
			return "", Result{}, fmt.Errorf("any requirement is nil")
		}
		return list.evaluateAny(typed.ID, typed.Requirements, entries)
	case AtLeast:
		return list.evaluateAtLeast(typed.ID, typed.MinCount, typed.Requirements, entries)
	case *AtLeast:
		if typed == nil {
			return "", Result{}, fmt.Errorf("at-least requirement is nil")
		}
		return list.evaluateAtLeast(typed.ID, typed.MinCount, typed.Requirements, entries)
	default:
		return "", Result{}, fmt.Errorf("unsupported requirement type %T", requirement)
	}
}

func (list List) evaluateAll(
	id string,
	requirements []Requirement,
	entries map[entryKey]transcript.Entry,
) (string, Result, error) {
	key, err := allKey(id)
	if err != nil {
		return "", Result{}, err
	}
	children, passed, err := list.evaluateRequirements(requirements, entries)
	if err != nil {
		return "", Result{}, err
	}
	return key, Result{Passed: passed, Requirements: children}, nil
}

func (list List) evaluateAny(
	id string,
	requirements []Requirement,
	entries map[entryKey]transcript.Entry,
) (string, Result, error) {
	key, err := anyKey(id)
	if err != nil {
		return "", Result{}, err
	}
	children, _, err := list.evaluateRequirements(requirements, entries)
	if err != nil {
		return "", Result{}, err
	}
	passed := false
	for _, child := range children {
		passed = passed || child.Passed
	}
	return key, Result{Passed: passed, Requirements: children}, nil
}

func (list List) evaluateAtLeast(
	id string,
	minCount int,
	requirements []Requirement,
	entries map[entryKey]transcript.Entry,
) (string, Result, error) {
	key, err := atLeastKey(id, minCount)
	if err != nil {
		return "", Result{}, err
	}
	children, _, err := list.evaluateRequirements(requirements, entries)
	if err != nil {
		return "", Result{}, err
	}
	passedCount := 0
	for _, child := range children {
		if child.Passed {
			passedCount++
		}
	}
	return key, Result{Passed: passedCount >= minCount, Requirements: children}, nil
}
