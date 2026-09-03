// Package semantic validates values shared across OPM model types.
package semantic

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

var (
	hostnamePattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*\.[a-z]{2,}$`)
	kebabCasePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	semverPattern    = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
)

// Email validates an email address without accepting a display name.
func Email(value string) error {
	address, err := mail.ParseAddress(value)
	if err != nil {
		return fmt.Errorf("invalid email address %q: %w", value, err)
	}
	if address.Address != value {
		return fmt.Errorf("invalid email address %q: display names are not allowed", value)
	}

	return nil
}

// Hostname validates the lowercase, dotted hostname form required by OPM.
func Hostname(value string) error {
	if len(value) > 253 || !hostnamePattern.MatchString(value) {
		return fmt.Errorf("invalid hostname %q", value)
	}

	labels := strings.Split(value, ".")
	for _, label := range labels {
		if len(label) > 63 {
			return fmt.Errorf("invalid hostname %q: label exceeds 63 characters", value)
		}
	}

	return nil
}

// KebabCase validates a lowercase kebab-case identifier.
func KebabCase(value string) error {
	if !kebabCasePattern.MatchString(value) {
		return fmt.Errorf("invalid kebab-case value %q", value)
	}

	return nil
}

// Semver validates a semantic version string.
func Semver(value string) error {
	if !semverPattern.MatchString(value) {
		return fmt.Errorf("invalid semantic version %q", value)
	}

	return nil
}

// UniqueStrings rejects repeated strings.
func UniqueStrings(values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate string %q", value)
		}
		seen[value] = struct{}{}
	}

	return nil
}

// URI validates an absolute RFC 3986 URI.
func URI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid URI %q: %w", value, err)
	}
	if !parsed.IsAbs() {
		return fmt.Errorf("invalid URI %q: scheme is required", value)
	}

	return nil
}
