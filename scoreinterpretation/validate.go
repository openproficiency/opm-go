package scoreinterpretation

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/openproficiency/opm-go/internal/schema"
	"github.com/openproficiency/opm-go/internal/semantic"
	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/topic"
)

// Validate checks interpretation metadata and the complete requirement tree.
func (interpretation Interpretation) Validate() error {
	return interpretation.validateWithTopics(make(map[string]struct{}))
}

func (interpretation Interpretation) validateWithTopics(topics map[string]struct{}) error {
	if err := semantic.KebabCase(interpretation.ID); err != nil {
		return fmt.Errorf("validate score interpretation ID: %w", err)
	}
	if interpretation.Name == "" {
		return errors.New("validate score interpretation: name is required")
	}
	if interpretation.Description == "" {
		return errors.New("validate score interpretation: description is required")
	}
	operators := make(map[string]struct{})
	if err := validateRequirements(interpretation.Requirements, operators, topics); err != nil {
		return fmt.Errorf("validate score interpretation %q: %w", interpretation.ID, err)
	}

	wire, err := makeWireInterpretation(interpretation, false)
	if err != nil {
		return fmt.Errorf("validate score interpretation %q: %w", interpretation.ID, err)
	}
	data, err := json.Marshal(wire)
	if err != nil {
		return fmt.Errorf("validate score interpretation %q: marshal JSON: %w", interpretation.ID, err)
	}
	if err := schema.ValidateJSON(schema.ScoreInterpretation, data); err != nil {
		return err
	}

	return nil
}

// Validate checks list metadata, dependencies, interpretations, and requirements.
func (list List) Validate() error {
	if list.IssuedAt.IsZero() {
		return errors.New("validate score interpretation list: issued-at timestamp is required")
	}
	if (list.signature == nil) != (list.signedBy == nil) {
		return errors.New("validate score interpretation list: signature and signed-by must both be null or both be set")
	}
	if list.signedBy != nil {
		if err := signerMatchesOwner(*list.signedBy, list.Owner); err != nil {
			return fmt.Errorf("validate score interpretation list: %w", err)
		}
	}

	if err := validateDependencies(list.Dependencies); err != nil {
		return err
	}
	qualifiedTopics := make(map[string]struct{})
	for key, interpretation := range list.Interpretations {
		if interpretation.ID != key {
			return fmt.Errorf(
				"validate score interpretation list: map key %q does not match interpretation ID %q",
				key,
				interpretation.ID,
			)
		}
		if err := interpretation.validateWithTopics(qualifiedTopics); err != nil {
			return err
		}
		if err := validateAliases(interpretation.Requirements, list.Dependencies); err != nil {
			return fmt.Errorf("validate score interpretation %q: %w", key, err)
		}
	}

	document, err := list.wireDocument(true)
	if err != nil {
		return err
	}
	document.Dependencies = nil
	data, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("validate score interpretation list: marshal JSON: %w", err)
	}
	if err := schema.ValidateJSON(schema.ScoreInterpretationList, data); err != nil {
		return err
	}

	return nil
}

func validateRequirements(
	requirements []Requirement,
	operators map[string]struct{},
	topics map[string]struct{},
) error {
	for _, requirement := range requirements {
		switch typed := requirement.(type) {
		case requiredTopic:
			if err := validateRequiredTopic(typed, topics); err != nil {
				return err
			}
		case All:
			if err := validateOperator(allKey, typed.ID, typed.Requirements, operators, topics); err != nil {
				return err
			}
		case *All:
			if typed == nil {
				return errors.New("all requirement is nil")
			}
			if err := validateOperator(allKey, typed.ID, typed.Requirements, operators, topics); err != nil {
				return err
			}
		case Any:
			if err := validateOperator(anyKey, typed.ID, typed.Requirements, operators, topics); err != nil {
				return err
			}
		case *Any:
			if typed == nil {
				return errors.New("any requirement is nil")
			}
			if err := validateOperator(anyKey, typed.ID, typed.Requirements, operators, topics); err != nil {
				return err
			}
		case AtLeast:
			if err := validateAtLeast(typed, operators, topics); err != nil {
				return err
			}
		case *AtLeast:
			if typed == nil {
				return errors.New("at-least requirement is nil")
			}
			if err := validateAtLeast(*typed, operators, topics); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported requirement type %T", requirement)
		}
	}

	return nil
}

func validateRequiredTopic(requirement requiredTopic, topics map[string]struct{}) error {
	if err := semantic.KebabCase(requirement.dependency); err != nil {
		return fmt.Errorf("invalid dependency alias: %w", err)
	}
	if err := semantic.KebabCase(requirement.topic); err != nil {
		return fmt.Errorf("invalid topic ID: %w", err)
	}
	if !validMinimum(requirement.minimum) {
		return fmt.Errorf("invalid minimum score %q", requirement.minimum.String())
	}
	if _, exists := topics[requirement.key()]; exists {
		return fmt.Errorf("duplicate qualified topic %q", requirement.key())
	}
	topics[requirement.key()] = struct{}{}
	return nil
}

func validateOperator(
	resolve func(string) (string, error),
	id string,
	requirements []Requirement,
	operators map[string]struct{},
	topics map[string]struct{},
) error {
	key, err := resolve(id)
	if err != nil {
		return err
	}
	if _, exists := operators[key]; exists {
		return fmt.Errorf("duplicate operator ID %q", key)
	}
	operators[key] = struct{}{}
	return validateRequirements(requirements, operators, topics)
}

func validateAtLeast(
	requirement AtLeast,
	operators map[string]struct{},
	topics map[string]struct{},
) error {
	key, err := atLeastKey(requirement.ID, requirement.MinCount)
	if err != nil {
		return err
	}
	if _, exists := operators[key]; exists {
		return fmt.Errorf("duplicate operator ID %q", key)
	}
	operators[key] = struct{}{}
	return validateRequirements(requirement.Requirements, operators, topics)
}

func validMinimum(value score.Score) bool {
	return value >= score.Unaware && value <= score.Fluent
}

func validateDependencies(dependencies map[string]topic.Dependency) error {
	for alias, dependency := range dependencies {
		if err := semantic.KebabCase(alias); err != nil {
			return fmt.Errorf("validate score interpretation dependency alias: %w", err)
		}
		if err := semantic.Hostname(dependency.Owner); err != nil {
			return fmt.Errorf("validate dependency %q owner: %w", alias, err)
		}
		if err := semantic.KebabCase(dependency.Name); err != nil {
			return fmt.Errorf("validate dependency %q name: %w", alias, err)
		}
		if err := semantic.Semver(dependency.Version); err != nil {
			return fmt.Errorf("validate dependency %q version: %w", alias, err)
		}
		for _, location := range dependency.Locations {
			if err := semantic.URI(location); err != nil {
				return fmt.Errorf("validate dependency %q location: %w", alias, err)
			}
		}
	}
	return nil
}

func validateAliases(requirements []Requirement, dependencies map[string]topic.Dependency) error {
	for _, requirement := range requirements {
		switch typed := requirement.(type) {
		case requiredTopic:
			if _, exists := dependencies[typed.dependency]; !exists {
				return fmt.Errorf("unknown dependency alias %q", typed.dependency)
			}
		case All:
			if err := validateAliases(typed.Requirements, dependencies); err != nil {
				return err
			}
		case *All:
			if typed != nil {
				if err := validateAliases(typed.Requirements, dependencies); err != nil {
					return err
				}
			}
		case Any:
			if err := validateAliases(typed.Requirements, dependencies); err != nil {
				return err
			}
		case *Any:
			if typed != nil {
				if err := validateAliases(typed.Requirements, dependencies); err != nil {
					return err
				}
			}
		case AtLeast:
			if err := validateAliases(typed.Requirements, dependencies); err != nil {
				return err
			}
		case *AtLeast:
			if typed != nil {
				if err := validateAliases(typed.Requirements, dependencies); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func signerMatchesOwner(email, owner string) error {
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return fmt.Errorf("invalid signer email %q", email)
	}
	at := strings.LastIndexByte(email, '@')
	if at < 0 || email[at+1:] != owner {
		return fmt.Errorf("signer email domain must exactly match owner %q", owner)
	}
	return nil
}
