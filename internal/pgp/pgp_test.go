package pgp

import (
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptUnlocksPrivateKeys(t *testing.T) {
	// Description
	// The configured passphrase unlocks an encrypted signing entity.

	// Arrange
	entity := newTestEntity(t, "proficiency@example.com")
	passphrase := "correct horse battery staple"
	encryptErr := entity.EncryptPrivateKeys([]byte(passphrase), nil)
	require.NoError(t, encryptErr)
	require.True(t, entity.PrivateKey.Encrypted)

	// Act
	err := Decrypt(entity, passphrase)

	// Assert
	require.NoError(t, err)
	assert.False(t, entity.PrivateKey.Encrypted)
}

func TestDecryptRejectsWrongPassphrase(t *testing.T) {
	// Description
	// An incorrect passphrase does not unlock an encrypted signing entity.

	// Arrange
	entity := newTestEntity(t, "proficiency@example.com")
	passphrase := "correct passphrase"
	wrongPassphrase := "incorrect passphrase"
	encryptErr := entity.EncryptPrivateKeys([]byte(passphrase), nil)
	require.NoError(t, encryptErr)

	// Act
	err := Decrypt(entity, wrongPassphrase)

	// Assert
	require.Error(t, err)
	assert.True(t, entity.PrivateKey.Encrypted)
}

func TestDecryptSupportsSubkeyOnlyPrivateKeys(t *testing.T) {
	// Description
	// A secret subkey export can be unlocked without a primary private key.

	// Arrange
	entity := newTestEntity(t, "proficiency@example.com")
	passphrase := "correct horse battery staple"
	encryptErr := entity.EncryptPrivateKeys([]byte(passphrase), nil)
	require.NoError(t, encryptErr)
	require.NotEmpty(t, entity.Subkeys)
	require.NotNil(t, entity.Subkeys[0].PrivateKey)
	entity.PrivateKey = nil

	// Act
	err := Decrypt(entity, passphrase)

	// Assert
	require.NoError(t, err)
	assert.False(t, entity.Subkeys[0].PrivateKey.Encrypted)
}

func TestSignAndVerifyDetachedArmor(t *testing.T) {
	// Description
	// Detached armor verifies against the original content and reports its signer.

	// Arrange
	entity := newTestEntity(t, "proficiency@example.com")
	message := []byte(`{"name":"math"}`)
	expectedEmail := "proficiency@example.com"

	// Act
	signature, signedBy, signErr := Sign(entity, message)
	verifiedBy, verifyErr := Verify(entity, message, signature)

	// Assert - Signing
	require.NoError(t, signErr)
	assert.Contains(t, signature, "-----BEGIN PGP SIGNATURE-----")
	assert.Equal(t, expectedEmail, signedBy)

	// Assert - Verification
	require.NoError(t, verifyErr)
	assert.Equal(t, expectedEmail, verifiedBy)
}

func TestVerifyRejectsChangedContent(t *testing.T) {
	// Description
	// A detached signature fails after its protected content changes.

	// Arrange
	entity := newTestEntity(t, "proficiency@example.com")
	original := []byte(`{"name":"math"}`)
	changed := []byte(`{"name":"science"}`)
	signature, _, signErr := Sign(entity, original)
	require.NoError(t, signErr)

	// Act
	verifiedBy, err := Verify(entity, changed, signature)

	// Assert
	require.Error(t, err)
	assert.Empty(t, verifiedBy)
}

func TestSignatureKeyIDReturnsSigningKey(t *testing.T) {
	// Description
	// The key ID embedded in detached armor identifies the selected signing key.

	// Arrange
	entity := newTestEntity(t, "proficiency@example.com")
	message := []byte("protected content")
	signingKey, signingKeyExists := entity.SigningKey(time.Now())
	require.True(t, signingKeyExists)
	expectedKeyID := signingKey.PublicKey.KeyId
	signature, _, signErr := Sign(entity, message)
	require.NoError(t, signErr)

	// Act
	keyID, err := SignatureKeyID(signature)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedKeyID, keyID)
}

func TestSignerEmailPrefersPrimaryIdentity(t *testing.T) {
	// Description
	// Signer metadata uses the primary identity instead of map ordering.

	// Arrange
	entity := newTestEntity(t, "primary@example.com")
	addIdentityErr := entity.AddUserId("Alternate", "", "alternate@example.com", testKeyConfig())
	require.NoError(t, addIdentityErr)
	expectedEmail := "primary@example.com"

	// Act
	email, err := SignerEmail(entity)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedEmail, email)
}

func newTestEntity(t *testing.T, email string) *openpgp.Entity {
	t.Helper()

	entity, err := openpgp.NewEntity("OPM Test", "", email, testKeyConfig())
	require.NoError(t, err)

	return entity
}

func testKeyConfig() *packet.Config {
	return &packet.Config{
		Algorithm: packet.PubKeyAlgoEdDSA,
	}
}
