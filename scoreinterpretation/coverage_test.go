package scoreinterpretation

import (
	"strings"
	"testing"

	"github.com/openproficiency/opm-go/score"
	"github.com/openproficiency/opm-go/topic"
	"github.com/openproficiency/opm-go/transcript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPointerOperatorsValidateEncodeAndEvaluate(t *testing.T) {
	// Description
	// Pointer forms of every public operator behave like their value forms.

	// Arrange
	list := validInterpretationList()
	list.Interpretations = map[string]Interpretation{
		"pointer-operators": {
			ID:          "pointer-operators",
			Name:        "Pointer Operators",
			Description: "Nested pointer operators.",
			Requirements: []Requirement{
				&All{
					ID: "foundations",
					Requirements: []Requirement{
						&Any{
							ID: "pathway",
							Requirements: []Requirement{
								&AtLeast{
									ID:       "breadth",
									MinCount: 2,
									Requirements: []Requirement{
										Require("math", "addition", score.Competent),
										Require("math", "subtraction", score.Aware),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	entries := []transcript.Entry{
		testEntry("math", "addition", score.Fluent, list.IssuedAt),
		testEntry("math", "subtraction", score.Competent, list.IssuedAt),
	}

	// Act
	validationErr := list.Validate()
	data, marshalErr := list.MarshalJSON()
	result, interpretErr := list.Interpret(entries, "pointer-operators")

	// Assert
	require.NoError(t, validationErr)
	require.NoError(t, marshalErr)
	require.NoError(t, interpretErr)
	assert.Contains(t, string(data), `"all-foundations"`)
	assert.True(t, result.Passed)
	allResult := result.Requirements["all-foundations"]
	assert.True(t, allResult.Passed)
	anyResult := allResult.Requirements["any-pathway"]
	assert.True(t, anyResult.Passed)
	assert.True(t, anyResult.Requirements["at-least-2-breadth"].Passed)
}

func TestInterpretationValidateRejectsNilAllPointer(t *testing.T) {
	// Description
	// A nil All pointer is rejected as an invalid public Requirement value.

	// Arrange
	interpretation := Interpretation{
		ID:           "nil-all",
		Name:         "Nil All",
		Description:  "Nil All pointer.",
		Requirements: []Requirement{(*All)(nil)},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all requirement is nil")
}

func TestInterpretationValidateRejectsNilAnyPointer(t *testing.T) {
	// Description
	// A nil Any pointer is rejected as an invalid public Requirement value.

	// Arrange
	interpretation := Interpretation{
		ID:           "nil-any",
		Name:         "Nil Any",
		Description:  "Nil Any pointer.",
		Requirements: []Requirement{(*Any)(nil)},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "any requirement is nil")
}

func TestInterpretationValidateRejectsNilAtLeastPointer(t *testing.T) {
	// Description
	// A nil AtLeast pointer is rejected as an invalid public Requirement value.

	// Arrange
	interpretation := Interpretation{
		ID:           "nil-at-least",
		Name:         "Nil At Least",
		Description:  "Nil AtLeast pointer.",
		Requirements: []Requirement{(*AtLeast)(nil)},
	}

	// Act
	err := interpretation.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at-least requirement is nil")
}

func TestListValidateRejectsUnknownAliasInsidePointerOperators(t *testing.T) {
	// Description
	// Alias validation traverses nested pointer operators.

	// Arrange
	list := validInterpretationList()
	list.Interpretations = map[string]Interpretation{
		"unknown-pointer-alias": {
			ID:          "unknown-pointer-alias",
			Name:        "Unknown Pointer Alias",
			Description: "Unknown alias nested in pointers.",
			Requirements: []Requirement{
				&All{
					ID: "outer",
					Requirements: []Requirement{
						&Any{
							ID: "middle",
							Requirements: []Requirement{
								&AtLeast{
									ID:           "inner",
									MinCount:     1,
									Requirements: []Requirement{Require("missing", "addition", score.Aware)},
								},
							},
						},
					},
				},
			},
		},
	}

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown dependency alias "missing"`)
}

func TestListValidateRejectsInvalidDependencyAlias(t *testing.T) {
	// Description
	// Dependency aliases must satisfy the schema kebab-case constraint.

	// Arrange
	list := validInterpretationList()
	list.Interpretations = map[string]Interpretation{}
	list.Dependencies["Invalid Alias"] = topic.Dependency{
		Owner:   "example.com",
		Name:    "math",
		Version: "0.1.0",
	}

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency alias")
}

func TestListValidateRejectsInvalidDependencyOwner(t *testing.T) {
	// Description
	// Dependency owners must be valid OPM hostnames.

	// Arrange
	list := validInterpretationList()
	list.Interpretations = map[string]Interpretation{}
	dependency := list.Dependencies["math"]
	dependency.Owner = "invalid"
	list.Dependencies["math"] = dependency

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `dependency "math" owner`)
}

func TestListValidateRejectsInvalidDependencyName(t *testing.T) {
	// Description
	// Dependency topic-list names must be kebab-case.

	// Arrange
	list := validInterpretationList()
	list.Interpretations = map[string]Interpretation{}
	dependency := list.Dependencies["math"]
	dependency.Name = "Invalid Name"
	list.Dependencies["math"] = dependency

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `dependency "math" name`)
}

func TestListValidateRejectsInvalidDependencyVersion(t *testing.T) {
	// Description
	// Dependency topic-list versions must use semantic version syntax.

	// Arrange
	list := validInterpretationList()
	list.Interpretations = map[string]Interpretation{}
	dependency := list.Dependencies["math"]
	dependency.Version = "latest"
	list.Dependencies["math"] = dependency

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `dependency "math" version`)
}

func TestListValidateRejectsInvalidDependencyLocation(t *testing.T) {
	// Description
	// Every dependency location must be an absolute URI.

	// Arrange
	list := validInterpretationList()
	list.Interpretations = map[string]Interpretation{}
	dependency := list.Dependencies["math"]
	dependency.Locations = []string{"relative/math.yml"}
	list.Dependencies["math"] = dependency

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `dependency "math" location`)
}

func TestListUnmarshalJSONRejectsMultipleValues(t *testing.T) {
	// Description
	// A JSON import accepts exactly one score interpretation list document.

	// Arrange
	list := validInterpretationList()
	data, marshalErr := list.MarshalJSON()
	require.NoError(t, marshalErr)
	data = append(data, []byte("\n{}")...)
	var decoded List

	// Act
	err := decoded.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.Empty(t, decoded.Name)
}

func TestListUnmarshalYAMLRejectsMultipleDocuments(t *testing.T) {
	// Description
	// A YAML import accepts exactly one score interpretation list document.

	// Arrange
	list := validInterpretationList()
	data, marshalErr := list.MarshalYAML()
	require.NoError(t, marshalErr)
	data = append(data, []byte("---\n{}\n")...)
	var decoded List

	// Act
	err := decoded.UnmarshalYAML(data)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple YAML documents")
	assert.Empty(t, decoded.Name)
}

func TestUnsignedListVerifyReturnsError(t *testing.T) {
	// Description
	// Verification rejects a list that has never been signed.

	// Arrange
	list := validInterpretationList()
	entity := newInterpretationTestEntity(t, "proficiency@example.com")

	// Act
	verified, err := list.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, verified)
}

func TestListVerifyRejectsDifferentPublicKey(t *testing.T) {
	// Description
	// A signature cannot be verified with an unrelated public key.

	// Arrange
	list := validInterpretationList()
	signer := newInterpretationTestEntity(t, "proficiency@example.com")
	other := newInterpretationTestEntity(t, "proficiency@example.com")
	require.NoError(t, list.Sign(signer, ""))

	// Act
	verified, err := list.Verify(other)

	// Assert
	require.Error(t, err)
	assert.False(t, verified)
}

func TestListVerifyRejectsMismatchedSignedByMetadata(t *testing.T) {
	// Description
	// Verification detects signed-by metadata that differs from the signature identity.

	// Arrange
	list := validInterpretationList()
	entity := newInterpretationTestEntity(t, "proficiency@example.com")
	require.NoError(t, list.Sign(entity, ""))
	data, marshalErr := list.MarshalJSON()
	require.NoError(t, marshalErr)
	corrupted := strings.Replace(
		string(data),
		`"signed-by":"proficiency@example.com"`,
		`"signed-by":"alternate@example.com"`,
		1,
	)
	var decoded List
	require.NoError(t, decoded.UnmarshalJSON([]byte(corrupted)))

	// Act
	verified, err := decoded.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, verified)
	assert.Contains(t, err.Error(), "differs from signed-by")
}
