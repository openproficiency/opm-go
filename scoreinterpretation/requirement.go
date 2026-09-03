// Package scoreinterpretation defines and evaluates OPM score interpretation lists.
package scoreinterpretation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/openproficiency/opm-go/score"
)

var (
	allOperatorPattern     = regexp.MustCompile(`^all(?:-[a-z][a-z0-9]*(?:-[a-z0-9]+)*)?$`)
	anyOperatorPattern     = regexp.MustCompile(`^any(?:-[a-z][a-z0-9]*(?:-[a-z0-9]+)*)?$`)
	atLeastOperatorPattern = regexp.MustCompile(`^at-least-([1-9][0-9]*)(?:-[a-z][a-z0-9]*(?:-[a-z0-9]+)*)?$`)
)

// Requirement is a topic score requirement or a logical group of requirements.
//
// The interface is sealed. Values are created with Require or with All, Any,
// and AtLeast.
type Requirement interface {
	isRequirement()
}

type requiredTopic struct {
	dependency string
	topic      string
	minimum    score.Score
}

// All requires every nested requirement to pass.
type All struct {
	ID           string
	Requirements []Requirement
}

// Any requires at least one nested requirement to pass.
type Any struct {
	ID           string
	Requirements []Requirement
}

// AtLeast requires MinCount nested requirements to pass.
type AtLeast struct {
	ID           string
	MinCount     int
	Requirements []Requirement
}

// Require creates one minimum score requirement for dependency.topic.
func Require(dependency, topicID string, minimum score.Score) Requirement {
	return requiredTopic{
		dependency: dependency,
		topic:      topicID,
		minimum:    minimum,
	}
}

func (requiredTopic) isRequirement() {}
func (All) isRequirement()           {}
func (Any) isRequirement()           {}
func (AtLeast) isRequirement()       {}

func (requirement requiredTopic) key() string {
	return requirement.dependency + "." + requirement.topic
}

func allKey(id string) (string, error) {
	return operatorKey("all", id, allOperatorPattern)
}

func anyKey(id string) (string, error) {
	return operatorKey("any", id, anyOperatorPattern)
}

func atLeastKey(id string, minCount int) (string, error) {
	if minCount <= 0 {
		return "", fmt.Errorf("at-least minimum count must be positive")
	}

	base := "at-least-" + strconv.Itoa(minCount)
	if id == base || strings.HasPrefix(id, base+"-") {
		if !atLeastOperatorPattern.MatchString(id) {
			return "", fmt.Errorf("invalid at-least operator ID %q", id)
		}
		return id, nil
	}
	if match := atLeastOperatorPattern.FindStringSubmatch(id); match != nil {
		return "", fmt.Errorf("at-least operator ID %q conflicts with minimum count %d", id, minCount)
	}
	if id == "" {
		return "", fmt.Errorf("at-least operator ID is required")
	}

	key := base + "-" + id
	if !atLeastOperatorPattern.MatchString(key) {
		return "", fmt.Errorf("invalid at-least operator ID %q", id)
	}
	return key, nil
}

func operatorKey(kind, id string, pattern *regexp.Regexp) (string, error) {
	if id == "" {
		return "", fmt.Errorf("%s operator ID is required", kind)
	}
	if id == kind || strings.HasPrefix(id, kind+"-") {
		if !pattern.MatchString(id) {
			return "", fmt.Errorf("invalid %s operator ID %q", kind, id)
		}
		return id, nil
	}

	key := kind + "-" + id
	if !pattern.MatchString(key) {
		return "", fmt.Errorf("invalid %s operator ID %q", kind, id)
	}
	return key, nil
}

func parseAtLeastCount(key string) (int, error) {
	match := atLeastOperatorPattern.FindStringSubmatch(key)
	if match == nil {
		return 0, fmt.Errorf("invalid at-least operator key %q", key)
	}

	count, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("parse at-least operator key %q: %w", key, err)
	}
	return count, nil
}
