package topic_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/openproficiency/opm-go/topic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAddUsesTopicID(t *testing.T) {
	// Arrange
	var list topic.List
	addition := topic.Topic{
		ID:          "addition",
		Description: "Combining quantities",
	}

	// Act
	list.Add(addition)

	// Assert
	require.Contains(t, list.Topics, "addition")
	assert.Equal(t, addition, list.Topics["addition"])
}

func TestListFullNameIncludesOwnerNameAndVersion(t *testing.T) {
	// Arrange
	list := validList()

	// Act
	fullName := list.FullName()

	// Assert
	assert.Equal(t, "example.com/math@0.1.0", fullName)
}

func TestListTopicSelectionsReturnNewMaps(t *testing.T) {
	// Arrange
	list := validList()

	// Act
	atomic := list.AtomicTopics()
	groups := list.GroupTopics()
	delete(atomic, "addition")

	// Assert
	assert.NotContains(t, atomic, "addition")
	assert.Contains(t, list.Topics, "addition")
	assert.Contains(t, groups, "arithmetic")
	assert.NotContains(t, groups, "addition")
}

func TestListTopicSelectionsFallBackToTopLevelTopicsForInvalidGraph(t *testing.T) {
	// Arrange
	list := validList()
	list.Topics["addition"] = topic.Topic{
		ID:          "mismatched-id",
		Description: "Combining quantities",
	}

	// Act
	atomic := list.AtomicTopics()
	groups := list.GroupTopics()
	validationErr := list.Validate()

	// Assert
	assert.Contains(t, atomic, "addition")
	assert.Contains(t, groups, "arithmetic")
	require.Error(t, validationErr)
	assert.Contains(t, validationErr.Error(), "does not match ID")
}

func TestListComplexityReportSupportsMethodExpression(t *testing.T) {
	// Arrange
	list := validList()
	reportFunction := topic.List.ComplexityReport

	// Act
	report := reportFunction(list)

	// Assert
	assert.Equal(t, 2, report.TopicCount)
	assert.Equal(t, 1, report.AtomicTopicCount)
	assert.Equal(t, 1, report.GroupTopicCount)
	assert.Equal(t, 1, report.MaxSubtopics)
	assert.Equal(t, 2, report.MaxDepth)
	assert.Equal(t, 1, report.DependenciesCount)
}

func TestListComplexityReportFallsBackToTopLevelTopicsForInvalidGraph(t *testing.T) {
	// Arrange
	list := validList()
	list.Topics["addition"] = topic.Topic{
		ID:          "mismatched-id",
		Description: "Combining quantities",
	}

	// Act
	report := list.ComplexityReport()
	validationErr := list.Validate()

	// Assert
	assert.Equal(t, 2, report.TopicCount)
	assert.Equal(t, 1, report.AtomicTopicCount)
	assert.Equal(t, 1, report.GroupTopicCount)
	assert.Equal(t, 1, report.MaxSubtopics)
	assert.Equal(t, 2, report.MaxDepth)
	assert.Equal(t, 1, report.DependenciesCount)
	require.Error(t, validationErr)
	assert.Contains(t, validationErr.Error(), "does not match ID")
}

func TestListJSONRoundTripPopulatesTopicIDs(t *testing.T) {
	// Arrange
	original := validList()

	// Act
	data, marshalErr := original.MarshalJSON()
	var decoded topic.List
	unmarshalErr := decoded.UnmarshalJSON(data)

	// Assert
	require.NoError(t, marshalErr)
	assert.NotContains(t, string(data), `"id":"addition"`)
	require.NoError(t, unmarshalErr)
	assert.Equal(t, "addition", decoded.Topics["addition"].ID)
	assert.Equal(t, original.Owner, decoded.Owner)
	assert.Equal(t, original.IssuedAt, decoded.IssuedAt)
}

func TestListJSONRoundTripPreservesSchemaAnnotations(t *testing.T) {
	// Arrange
	data := []byte(`{
		"$schema":"https://example.com/topic-list.schema.json",
		"owner":"example.com",
		"name":"math",
		"description":"Basic mathematics",
		"version":"0.1.0",
		"issued-at":"2026-09-01T00:00:00Z",
		"signature":null,
		"signed-by":null,
		"topics":{"addition":{
			"$schema":"https://example.com/topic.schema.json",
			"description":"Combining quantities"
		}}
	}`)
	var list topic.List

	// Act
	unmarshalErr := list.UnmarshalJSON(data)
	encoded, marshalErr := list.MarshalJSON()

	// Assert
	require.NoError(t, unmarshalErr)
	require.NoError(t, marshalErr)
	assert.Contains(t, string(encoded), `"$schema":"https://example.com/topic-list.schema.json"`)
	assert.Contains(t, string(encoded), `"$schema":"https://example.com/topic.schema.json"`)
}

func TestListJSONOutputIsDeterministic(t *testing.T) {
	// Arrange
	list := validList()

	// Act
	first, firstErr := list.MarshalJSON()
	second, secondErr := list.MarshalJSON()

	// Assert
	require.NoError(t, firstErr)
	require.NoError(t, secondErr)
	assert.Equal(t, first, second)
	assert.Contains(t, string(first), `"signature":null,"signed-by":null`)
	assert.Less(t, strings.Index(string(first), `"addition"`), strings.Index(string(first), `"arithmetic"`))
}

func TestListYAMLRoundTripUsesOPMFieldNames(t *testing.T) {
	// Arrange
	original := validList()

	// Act
	data, marshalErr := original.MarshalYAML()
	var decoded topic.List
	unmarshalErr := decoded.UnmarshalYAML(data)

	// Assert
	require.NoError(t, marshalErr)
	assert.Contains(t, string(data), "issued-at: 2026-09-01T00:00:00Z")
	assert.Contains(t, string(data), "signed-by: null")
	assert.Contains(t, string(data), "display-name: Addition")
	require.NoError(t, unmarshalErr)
	assert.Equal(t, "addition", decoded.Topics["addition"].ID)
	assert.Equal(t, "example.com", decoded.Dependencies["standard"].Owner)
}

func TestListDependencyShorthandRoundTrip(t *testing.T) {
	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"binary-math",
		"description":"Binary mathematics",
		"version":"0.1.0",
		"issued-at":"2026-09-01T00:00:00Z",
		"signature":null,
		"signed-by":null,
		"topics":{"binary-addition":{"description":"Binary addition"}},
		"dependencies":{"std-math":"example.com/math@0.1.0"}
	}`)
	var list topic.List

	// Act
	unmarshalErr := list.UnmarshalJSON(data)
	encoded, marshalErr := list.MarshalJSON()

	// Assert
	require.NoError(t, unmarshalErr)
	assert.Equal(t, topic.Dependency{Owner: "example.com", Name: "math", Version: "0.1.0"}, list.Dependencies["std-math"])
	require.NoError(t, marshalErr)
	assert.Contains(t, string(encoded), `"std-math":"example.com/math@0.1.0"`)
}

func TestListDependencyLongFormRoundTrip(t *testing.T) {
	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"binary-math",
		"description":"Binary mathematics",
		"version":"0.1.0",
		"issued-at":"2026-09-01T00:00:00Z",
		"signature":null,
		"signed-by":null,
		"topics":{"binary-addition":{"description":"Binary addition"}},
		"dependencies":{"std-math":{
			"topic-list-owner":"example.com",
			"topic-list-name":"math",
			"topic-list-version":"0.1.0",
			"locations":["https://example.com/math.yml"]
		}}
	}`)
	var list topic.List

	// Act
	unmarshalErr := list.UnmarshalJSON(data)
	encoded, marshalErr := list.MarshalJSON()

	// Assert
	require.NoError(t, unmarshalErr)
	assert.Equal(t, []string{"https://example.com/math.yml"}, list.Dependencies["std-math"].Locations)
	require.NoError(t, marshalErr)
	assert.Contains(t, string(encoded), `"topic-list-owner":"example.com"`)
	assert.Contains(t, string(encoded), `"locations":["https://example.com/math.yml"]`)
}

func TestListDependencyLongFormWithoutLocationsRemainsLong(t *testing.T) {
	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"binary-math",
		"description":"Binary mathematics",
		"version":"0.1.0",
		"issued-at":"2026-09-01T00:00:00Z",
		"signature":null,
		"signed-by":null,
		"topics":{"binary-addition":{"description":"Binary addition"}},
		"dependencies":{"std-math":{
			"topic-list-owner":"example.com",
			"topic-list-name":"math",
			"topic-list-version":"0.1.0"
		}}
	}`)
	var list topic.List

	// Act
	unmarshalErr := list.UnmarshalJSON(data)
	encoded, marshalErr := list.MarshalJSON()

	// Assert
	require.NoError(t, unmarshalErr)
	require.NoError(t, marshalErr)
	assert.Contains(t, string(encoded), `"std-math":{"topic-list-owner":"example.com"`)
	assert.NotContains(t, string(encoded), `"locations"`)
}

func TestListInlineTopicRoundTripPreservesNestedObject(t *testing.T) {
	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"math",
		"description":"Basic mathematics",
		"version":"0.1.0",
		"issued-at":"2026-09-01T00:00:00Z",
		"signature":null,
		"signed-by":null,
		"topics":{"arithmetic":{
			"description":"Arithmetic",
			"subtopics":[{"id":"addition","description":"Combining quantities"}]
		}}
	}`)
	var list topic.List

	// Act
	unmarshalErr := list.UnmarshalJSON(data)
	encoded, marshalErr := list.MarshalJSON()
	report := list.ComplexityReport()

	// Assert
	require.NoError(t, unmarshalErr)
	require.NoError(t, marshalErr)
	assert.Contains(t, string(encoded), `"subtopics":[{"id":"addition","description":"Combining quantities"}]`)
	assert.Equal(t, []string{"addition"}, list.Topics["arithmetic"].Subtopics)
	assert.Equal(t, 2, report.TopicCount)
	assert.Equal(t, 2, report.MaxDepth)
}

func TestListValidateAcceptsValidGraph(t *testing.T) {
	// Arrange
	list := validList()

	// Act
	err := list.Validate()

	// Assert
	require.NoError(t, err)
}

func TestListValidateRejectsMapKeyIDMismatch(t *testing.T) {
	// Arrange
	list := validList()
	list.Topics["addition"] = topic.Topic{
		ID:          "subtraction",
		Description: "Combining quantities",
	}

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match ID")
}

func TestListMarshalRejectsMissingIssuedAt(t *testing.T) {
	// Arrange
	list := validList()
	list.IssuedAt = time.Time{}

	// Act
	jsonData, jsonErr := list.MarshalJSON()
	yamlData, yamlErr := list.MarshalYAML()

	// Assert
	assert.Nil(t, jsonData)
	require.Error(t, jsonErr)
	assert.Contains(t, jsonErr.Error(), "issued-at timestamp is required")
	assert.Nil(t, yamlData)
	require.Error(t, yamlErr)
	assert.Contains(t, yamlErr.Error(), "issued-at timestamp is required")
}

func TestListValidateRejectsUnknownSubtopic(t *testing.T) {
	// Arrange
	list := validList()
	list.Topics["arithmetic"] = topic.Topic{
		ID:          "arithmetic",
		Description: "Arithmetic",
		Subtopics:   []string{"missing"},
	}

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown subtopic "missing"`)
}

func TestListValidateRejectsSharedSubtopic(t *testing.T) {
	// Arrange
	list := validList()
	list.Topics["basic-math"] = topic.Topic{
		ID:          "basic-math",
		Description: "Basic math",
		Subtopics:   []string{"addition"},
	}

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `subtopic "addition" is shared`)
}

func TestListValidateRejectsSubtopicCycle(t *testing.T) {
	// Arrange
	list := validList()
	list.Topics["addition"] = topic.Topic{
		ID:          "addition",
		Description: "Combining quantities",
		Subtopics:   []string{"arithmetic"},
	}

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subtopic cycle")
}

func TestListValidateRejectsUnknownLocalPretopic(t *testing.T) {
	// Arrange
	list := validList()
	addition := list.Topics["addition"]
	addition.Pretopics = []string{"numbers"}
	list.Topics["addition"] = addition

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown pretopic "numbers"`)
}

func TestListValidateAcceptsExternalPretopic(t *testing.T) {
	// Arrange
	list := validList()
	addition := list.Topics["addition"]
	addition.Pretopics = []string{"standard.numbers"}
	list.Topics["addition"] = addition

	// Act
	err := list.Validate()

	// Assert
	require.NoError(t, err)
}

func TestListValidateRejectsUnknownDependencyAlias(t *testing.T) {
	// Arrange
	list := validList()
	addition := list.Topics["addition"]
	addition.Pretopics = []string{"unknown.numbers"}
	list.Topics["addition"] = addition

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown dependency alias "unknown"`)
}

func TestListValidateRejectsPretopicCycle(t *testing.T) {
	// Arrange
	list := validList()
	addition := list.Topics["addition"]
	addition.Pretopics = []string{"arithmetic"}
	list.Topics["addition"] = addition
	arithmetic := list.Topics["arithmetic"]
	arithmetic.Pretopics = []string{"addition"}
	list.Topics["arithmetic"] = arithmetic

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pretopic cycle")
}

func TestListValidateRejectsExcessiveDepth(t *testing.T) {
	// Arrange
	list := topic.List{
		Owner:       "example.com",
		Name:        "deep-topics",
		Description: "Deep topic hierarchy",
		Version:     "0.1.0",
		IssuedAt:    testTime(),
		Topics:      make(map[string]topic.Topic),
	}
	for index := 0; index < 101; index++ {
		id := fmt.Sprintf("topic-%03d", index)
		current := topic.Topic{ID: id, Description: "Topic"}
		if index < 100 {
			current.Subtopics = []string{fmt.Sprintf("topic-%03d", index+1)}
		}
		list.Topics[id] = current
	}

	// Act
	err := list.Validate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum subtopic depth 100")
}

func TestListSignAndVerify(t *testing.T) {
	// Arrange
	list := validList()
	entity := newTestEntity(t, "proficiency@example.com")

	// Act
	signErr := list.Sign(entity, "")
	signature, signatureErr := list.Signature()
	signedBy, signedByErr := list.SignedBy()
	keyID, keyIDErr := list.SignatureKeyID()
	valid, verifyErr := list.Verify(entity)

	// Assert
	require.NoError(t, signErr)
	require.NoError(t, signatureErr)
	assert.Contains(t, signature, "-----BEGIN PGP SIGNATURE-----")
	require.NoError(t, signedByErr)
	assert.Equal(t, "proficiency@example.com", signedBy)
	require.NoError(t, keyIDErr)
	assert.NotZero(t, keyID)
	require.NoError(t, verifyErr)
	assert.True(t, valid)
}

func TestListSignRelocksEncryptedKey(t *testing.T) {
	// Arrange
	list := validList()
	entity := newTestEntity(t, "proficiency@example.com")
	passphrase := "correct horse battery staple"
	encryptErr := entity.EncryptPrivateKeys([]byte(passphrase), nil)
	require.NoError(t, encryptErr)
	require.True(t, entity.PrivateKey.Encrypted)

	// Act
	err := list.Sign(entity, passphrase)

	// Assert
	require.NoError(t, err)
	assert.True(t, entity.PrivateKey.Encrypted)
}

func TestListSignChecksPassphraseAfterEarlierSuccess(t *testing.T) {
	// Arrange
	list := validList()
	entity := newTestEntity(t, "proficiency@example.com")
	passphrase := "correct horse battery staple"
	wrongPassphrase := "incorrect passphrase"
	encryptErr := entity.EncryptPrivateKeys([]byte(passphrase), nil)
	require.NoError(t, encryptErr)
	firstSignErr := list.Sign(entity, passphrase)
	require.NoError(t, firstSignErr)
	require.True(t, entity.PrivateKey.Encrypted)

	// Act
	err := list.Sign(entity, wrongPassphrase)

	// Assert
	require.Error(t, err)
	assert.True(t, entity.PrivateKey.Encrypted)
}

func TestListSignRejectsSignerOutsideOwnerDomain(t *testing.T) {
	// Arrange
	list := validList()
	entity := newTestEntity(t, "proficiency@other.example")

	// Act
	err := list.Sign(entity, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `must exactly match owner "example.com"`)
}

func TestListSignatureBecomesStaleAfterProtectedChange(t *testing.T) {
	// Arrange
	list := validList()
	entity := newTestEntity(t, "proficiency@example.com")
	require.NoError(t, list.Sign(entity, ""))
	list.Description = "Changed mathematics"

	// Act
	signature, signatureErr := list.Signature()
	signedBy, signedByErr := list.SignedBy()
	keyID, keyIDErr := list.SignatureKeyID()
	valid, verifyErr := list.Verify(entity)
	jsonData, jsonErr := list.MarshalJSON()
	yamlData, yamlErr := list.MarshalYAML()

	// Assert
	assert.Empty(t, signature)
	assert.ErrorIs(t, signatureErr, topic.ErrSignatureStale)
	assert.Empty(t, signedBy)
	assert.ErrorIs(t, signedByErr, topic.ErrSignatureStale)
	assert.Zero(t, keyID)
	assert.ErrorIs(t, keyIDErr, topic.ErrSignatureStale)
	assert.False(t, valid)
	assert.ErrorIs(t, verifyErr, topic.ErrSignatureStale)
	require.NoError(t, jsonErr)
	assert.Contains(t, string(jsonData), `"signature":null,"signed-by":null`)
	require.NoError(t, yamlErr)
	assert.Contains(t, string(yamlData), "signature: null\nsigned-by: null")
}

func TestListDependencyLocationsDoNotStaleSignature(t *testing.T) {
	// Arrange
	list := validList()
	entity := newTestEntity(t, "proficiency@example.com")
	require.NoError(t, list.Sign(entity, ""))
	dependency := list.Dependencies["standard"]
	dependency.Locations = []string{"https://mirror.example.com/math.yml"}
	list.Dependencies["standard"] = dependency

	// Act
	signature, signatureErr := list.Signature()
	valid, verifyErr := list.Verify(entity)

	// Assert
	require.NoError(t, signatureErr)
	assert.NotEmpty(t, signature)
	require.NoError(t, verifyErr)
	assert.True(t, valid)
}

func TestListSignedJSONRoundTripRetainsVerifiableSignature(t *testing.T) {
	// Arrange
	original := validList()
	entity := newTestEntity(t, "proficiency@example.com")
	require.NoError(t, original.Sign(entity, ""))

	// Act
	data, marshalErr := original.MarshalJSON()
	var decoded topic.List
	unmarshalErr := decoded.UnmarshalJSON(data)
	valid, verifyErr := decoded.Verify(entity)

	// Assert
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	require.NoError(t, verifyErr)
	assert.True(t, valid)
}

func TestListVerifyRejectsDifferentPublicKey(t *testing.T) {
	// Arrange
	list := validList()
	signer := newTestEntity(t, "proficiency@example.com")
	other := newTestEntity(t, "proficiency@example.com")
	require.NoError(t, list.Sign(signer, ""))

	// Act
	valid, err := list.Verify(other)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
}

func TestListVerifyRejectsSignedByMetadataMismatch(t *testing.T) {
	// Arrange
	original := validList()
	entity := newTestEntity(t, "proficiency@example.com")
	require.NoError(t, original.Sign(entity, ""))
	data, marshalErr := original.MarshalJSON()
	require.NoError(t, marshalErr)
	tampered := strings.Replace(
		string(data),
		`"signed-by":"proficiency@example.com"`,
		`"signed-by":"alternate@example.com"`,
		1,
	)
	var decoded topic.List
	require.NoError(t, decoded.UnmarshalJSON([]byte(tampered)))

	// Act
	valid, err := decoded.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "differs from signed-by")
	assert.False(t, valid)
}

func TestListUnmarshalRejectsTopicIDMismatch(t *testing.T) {
	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"math",
		"description":"Basic mathematics",
		"version":"0.1.0",
		"issued-at":"2026-09-01T00:00:00Z",
		"signature":null,
		"signed-by":null,
		"topics":{"addition":{"id":"subtraction","description":"Combining quantities"}}
	}`)
	var list topic.List

	// Act
	err := list.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `ID "subtraction" does not match map key`)
}

func TestListUnmarshalRejectsInlineTopicWithoutID(t *testing.T) {
	// Arrange
	data := []byte(`{
		"owner":"example.com",
		"name":"math",
		"description":"Basic mathematics",
		"version":"0.1.0",
		"issued-at":"2026-09-01T00:00:00Z",
		"signature":null,
		"signed-by":null,
		"topics":{"arithmetic":{"description":"Arithmetic","subtopics":[{"description":"Addition"}]}}
	}`)
	var list topic.List

	// Act
	err := list.UnmarshalJSON(data)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ID is required")
}

func TestUnsignedListSignatureMethodsReturnErrors(t *testing.T) {
	// Arrange
	list := validList()

	// Act
	signature, signatureErr := list.Signature()
	signedBy, signedByErr := list.SignedBy()
	keyID, keyIDErr := list.SignatureKeyID()
	valid, verifyErr := list.Verify(newTestEntity(t, "proficiency@example.com"))

	// Assert
	assert.Empty(t, signature)
	require.Error(t, signatureErr)
	assert.False(t, errors.Is(signatureErr, topic.ErrSignatureStale))
	assert.Empty(t, signedBy)
	require.Error(t, signedByErr)
	assert.Zero(t, keyID)
	require.Error(t, keyIDErr)
	assert.False(t, valid)
	require.Error(t, verifyErr)
}

func validList() topic.List {
	return topic.List{
		Owner:       "example.com",
		Name:        "math",
		Description: "Basic mathematics",
		Version:     "0.1.0",
		IssuedAt:    testTime(),
		Topics: map[string]topic.Topic{
			"addition": {
				ID:             "addition",
				DisplayName:    "Addition",
				Description:    "Combining quantities",
				DocsURL:        "https://example.com/docs/addition",
				ValidityPeriod: 732,
			},
			"arithmetic": {
				ID:          "arithmetic",
				Description: "Arithmetic operations",
				Subtopics:   []string{"addition"},
			},
		},
		Dependencies: map[string]topic.Dependency{
			"standard": {
				Owner:   "example.com",
				Name:    "standard-math",
				Version: "0.1.0",
			},
		},
	}
}

func testTime() time.Time {
	return time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
}

func newTestEntity(t *testing.T, email string) *openpgp.Entity {
	t.Helper()

	entity, err := openpgp.NewEntity("OPM Test", "", email, &packet.Config{
		Algorithm: packet.PubKeyAlgoEdDSA,
	})
	require.NoError(t, err)

	return entity
}

func TestListStandardJSONIntegration(t *testing.T) {
	// Arrange
	original := validList()

	// Act
	data, marshalErr := json.Marshal(original)
	var decoded topic.List
	unmarshalErr := json.Unmarshal(data, &decoded)

	// Assert
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	assert.Equal(t, original.FullName(), decoded.FullName())
}
