package scoreinterpretation

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSignExposesMetadataAndVerifies(t *testing.T) {
	// Description
	// A valid encrypted key signs the list and exposes verified signer metadata.

	// Arrange
	list := validInterpretationList()
	entity := newInterpretationTestEntity(t, "proficiency@example.com")
	passphrase := "correct horse battery staple"
	require.NoError(t, entity.EncryptPrivateKeys([]byte(passphrase), nil))

	// Act
	signErr := list.Sign(entity, passphrase)
	signature, signatureErr := list.Signature()
	signedBy, signedByErr := list.SignedBy()
	keyID, keyIDErr := list.SignatureKeyID()
	verified, verifyErr := list.Verify(entity)

	// Assert
	require.NoError(t, signErr)
	require.NoError(t, signatureErr)
	require.NoError(t, signedByErr)
	require.NoError(t, keyIDErr)
	require.NoError(t, verifyErr)
	assert.Contains(t, signature, "BEGIN PGP SIGNATURE")
	assert.Equal(t, "proficiency@example.com", signedBy)
	assert.NotZero(t, keyID)
	assert.True(t, verified)
	assert.True(t, entity.PrivateKey.Encrypted)
}

func TestListSigningRoundTripPreservesVerification(t *testing.T) {
	// Description
	// Signed JSON captures fresh signature state when decoded.

	// Arrange
	list := validInterpretationList()
	entity := newInterpretationTestEntity(t, "proficiency@example.com")
	require.NoError(t, list.Sign(entity, ""))
	var decoded List

	// Act
	data, marshalErr := list.MarshalJSON()
	unmarshalErr := decoded.UnmarshalJSON(data)
	verified, verifyErr := decoded.Verify(entity)

	// Assert
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	require.NoError(t, verifyErr)
	assert.True(t, verified)
}

func TestListSignatureBecomesStaleAfterProtectedMutation(t *testing.T) {
	// Description
	// Changing protected interpretation content invalidates all signature metadata APIs.

	// Arrange
	list := validInterpretationList()
	entity := newInterpretationTestEntity(t, "proficiency@example.com")
	require.NoError(t, list.Sign(entity, ""))
	interpretation := list.Interpretations["arithmetic"]
	interpretation.Description = "Changed"
	list.Interpretations["arithmetic"] = interpretation

	// Act
	signature, signatureErr := list.Signature()
	signedBy, signedByErr := list.SignedBy()
	keyID, keyIDErr := list.SignatureKeyID()
	verified, verifyErr := list.Verify(entity)

	// Assert
	assert.Empty(t, signature)
	assert.Empty(t, signedBy)
	assert.Zero(t, keyID)
	assert.False(t, verified)
	assert.ErrorIs(t, signatureErr, ErrSignatureStale)
	assert.ErrorIs(t, signedByErr, ErrSignatureStale)
	assert.ErrorIs(t, keyIDErr, ErrSignatureStale)
	assert.ErrorIs(t, verifyErr, ErrSignatureStale)
}

func TestListDependencyLocationsDoNotStaleSignature(t *testing.T) {
	// Description
	// Dependency locations are mutable distribution hints excluded from protected content.

	// Arrange
	list := validInterpretationList()
	entity := newInterpretationTestEntity(t, "proficiency@example.com")
	require.NoError(t, list.Sign(entity, ""))
	dependency := list.Dependencies["math"]
	dependency.Locations = []string{"https://mirror.example.com/math.yml"}
	list.Dependencies["math"] = dependency

	// Act
	verified, err := list.Verify(entity)

	// Assert
	require.NoError(t, err)
	assert.True(t, verified)
}

func TestListMarshalJSONEmitsNullMetadataWhenUnsignedOrStale(t *testing.T) {
	// Description
	// Unsigned and stale lists encode both signature metadata fields as null.

	// Arrange
	unsigned := validInterpretationList()
	signed := validInterpretationList()
	entity := newInterpretationTestEntity(t, "proficiency@example.com")
	require.NoError(t, signed.Sign(entity, ""))
	signed.Description = "Changed after signing"

	// Act
	unsignedData, unsignedErr := unsigned.MarshalJSON()
	staleData, staleErr := signed.MarshalJSON()
	unsignedYAML, unsignedYAMLErr := unsigned.MarshalYAML()
	staleYAML, staleYAMLErr := signed.MarshalYAML()

	// Assert
	require.NoError(t, unsignedErr)
	require.NoError(t, staleErr)
	require.NoError(t, unsignedYAMLErr)
	require.NoError(t, staleYAMLErr)
	var unsignedDocument map[string]any
	var staleDocument map[string]any
	require.NoError(t, json.Unmarshal(unsignedData, &unsignedDocument))
	require.NoError(t, json.Unmarshal(staleData, &staleDocument))
	assert.Nil(t, unsignedDocument["signature"])
	assert.Nil(t, unsignedDocument["signed-by"])
	assert.Nil(t, staleDocument["signature"])
	assert.Nil(t, staleDocument["signed-by"])
	assert.Contains(t, string(unsignedYAML), "signature: null")
	assert.Contains(t, string(unsignedYAML), "signed-by: null")
	assert.Contains(t, string(staleYAML), "signature: null")
	assert.Contains(t, string(staleYAML), "signed-by: null")
}

func TestListSignRejectsWrongPassphrase(t *testing.T) {
	// Description
	// An encrypted signing key cannot be used with an incorrect passphrase.

	// Arrange
	list := validInterpretationList()
	entity := newInterpretationTestEntity(t, "proficiency@example.com")
	require.NoError(t, entity.EncryptPrivateKeys([]byte("correct passphrase"), nil))

	// Act
	err := list.Sign(entity, "wrong passphrase")

	// Assert
	require.Error(t, err)
	_, signatureErr := list.Signature()
	require.Error(t, signatureErr)
	assert.False(t, errors.Is(signatureErr, ErrSignatureStale))
}

func TestListSignValidatesBeforeUsingKey(t *testing.T) {
	// Description
	// Invalid list content is rejected before signing is attempted.

	// Arrange
	list := validInterpretationList()
	list.Owner = "INVALID"

	// Act
	err := list.Sign(nil, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hostname")
}

func TestListSignRejectsSignerFromDifferentOwner(t *testing.T) {
	// Description
	// The signing identity email domain must exactly match the list owner.

	// Arrange
	list := validInterpretationList()
	entity := newInterpretationTestEntity(t, "proficiency@other.example")

	// Act
	err := list.Sign(entity, "")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), `must exactly match owner "example.com"`)
}

func newInterpretationTestEntity(t *testing.T, email string) *openpgp.Entity {
	t.Helper()
	entity, err := openpgp.NewEntity("OPM Interpretation Test", "", email, &packet.Config{
		Algorithm: packet.PubKeyAlgoEdDSA,
	})
	require.NoError(t, err)
	return entity
}
