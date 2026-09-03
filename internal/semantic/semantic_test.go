package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailAcceptsMailbox(t *testing.T) {
	// Description
	// A plain mailbox address is valid OPM email syntax.

	// Arrange
	value := "learner@example.com"

	// Act
	err := Email(value)

	// Assert
	require.NoError(t, err)
}

func TestEmailRejectsDisplayName(t *testing.T) {
	// Description
	// Display names are not valid where an OPM field requires only an address.

	// Arrange
	value := "Learner <learner@example.com>"

	// Act
	err := Email(value)

	// Assert
	require.Error(t, err)
}

func TestURIAcceptsFragment(t *testing.T) {
	// Description
	// An absolute URI may identify a section with a fragment.

	// Arrange
	value := "https://example.com/docs#introduction"

	// Act
	err := URI(value)

	// Assert
	require.NoError(t, err)
}

func TestHostnameAcceptsOwnerDomain(t *testing.T) {
	// Description
	// A lowercase registrable domain is valid OPM owner syntax.

	// Arrange
	value := "example.com"

	// Act
	err := Hostname(value)

	// Assert
	require.NoError(t, err)
}

func TestHostnameRejectsSubdomain(t *testing.T) {
	// Description
	// The v0.1.1 owner pattern accepts exactly one dotted domain suffix.

	// Arrange
	value := "learning.example.com"

	// Act
	err := Hostname(value)

	// Assert
	require.Error(t, err)
}

func TestKebabCaseRejectsUppercase(t *testing.T) {
	// Description
	// OPM identifiers must use lowercase kebab-case.

	// Arrange
	value := "Math-Level"

	// Act
	err := KebabCase(value)

	// Assert
	require.Error(t, err)
}

func TestSemverAcceptsPrereleaseAndBuild(t *testing.T) {
	// Description
	// OPM versions support semantic-version prerelease and build fields.

	// Arrange
	value := "1.2.3-rc.1+build.7"

	// Act
	err := Semver(value)

	// Assert
	require.NoError(t, err)
}

func TestSemverRejectsLeadingZero(t *testing.T) {
	// Description
	// Semantic-version numeric components cannot contain leading zeroes.

	// Arrange
	value := "01.2.3"

	// Act
	err := Semver(value)

	// Assert
	require.Error(t, err)
}

func TestUniqueStringsRejectsDuplicate(t *testing.T) {
	// Description
	// Repeated values are reported with the duplicated string.

	// Arrange
	values := []string{"one", "two", "one"}
	expectedMessage := `duplicate string "one"`

	// Act
	err := UniqueStrings(values)

	// Assert
	require.EqualError(t, err, expectedMessage)
}

func TestURIAllowsAbsoluteCustomScheme(t *testing.T) {
	// Description
	// Absolute URIs may use a non-HTTP scheme such as an OPM package location.

	// Arrange
	value := "npm:@example/math@1.0.0"

	// Act
	err := URI(value)

	// Assert
	require.NoError(t, err)
}

func TestURIRejectsRelativeReference(t *testing.T) {
	// Description
	// OPM URI fields cannot contain relative references.

	// Arrange
	value := "schemas/topic.json"

	// Act
	err := URI(value)

	// Assert
	assert.Error(t, err)
}
