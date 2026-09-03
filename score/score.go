// Package score defines the ordered Open Proficiency Model score levels.
package score

import (
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ErrInvalidScore reports a value outside the five OPM score levels.
var ErrInvalidScore = errors.New("invalid score")

// Score is an ordered Open Proficiency Model proficiency level.
type Score int

const (
	// Unaware indicates no awareness of the topic.
	Unaware Score = iota
	// Aware indicates awareness of the topic.
	Aware
	// Familiar indicates familiarity with the topic.
	Familiar
	// Competent indicates practical competence in the topic.
	Competent
	// Fluent indicates fluent proficiency in the topic.
	Fluent
)

var scoreNames = [...]string{
	"unaware",
	"aware",
	"familiar",
	"competent",
	"fluent",
}

// String returns the lowercase OPM wire value.
func (score Score) String() string {
	if !score.valid() {
		return ""
	}

	return scoreNames[score]
}

// MarshalJSON encodes a score as its lowercase OPM string.
func (score Score) MarshalJSON() ([]byte, error) {
	if !score.valid() {
		return nil, fmt.Errorf("%w: %d", ErrInvalidScore, score)
	}

	return json.Marshal(score.String())
}

// UnmarshalJSON decodes an exact lowercase OPM score string.
func (score *Score) UnmarshalJSON(data []byte) error {
	if score == nil {
		return errors.New("unmarshal score: nil receiver")
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("%w: decode JSON: %v", ErrInvalidScore, err)
	}

	parsed, err := parse(value)
	if err != nil {
		return err
	}

	*score = parsed
	return nil
}

// MarshalYAML encodes a score as its lowercase OPM string.
func (score Score) MarshalYAML() (any, error) {
	if !score.valid() {
		return nil, fmt.Errorf("%w: %d", ErrInvalidScore, score)
	}

	return score.String(), nil
}

// UnmarshalYAML decodes an exact lowercase OPM score string.
func (score *Score) UnmarshalYAML(node *yaml.Node) error {
	if score == nil {
		return errors.New("unmarshal score: nil receiver")
	}
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return fmt.Errorf("%w: YAML value must be a string", ErrInvalidScore)
	}

	parsed, err := parse(node.Value)
	if err != nil {
		return err
	}

	*score = parsed
	return nil
}

func (score Score) valid() bool {
	return score >= Unaware && score <= Fluent
}

func parse(value string) (Score, error) {
	switch value {
	case "unaware":
		return Unaware, nil
	case "aware":
		return Aware, nil
	case "familiar":
		return Familiar, nil
	case "competent":
		return Competent, nil
	case "fluent":
		return Fluent, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrInvalidScore, value)
	}
}
