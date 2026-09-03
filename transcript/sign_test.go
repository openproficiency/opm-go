package transcript

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/openproficiency/opm-go/internal/canonical"
	"github.com/openproficiency/opm-go/internal/pgp"
	"github.com/openproficiency/opm-go/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntrySignAndVerify(t *testing.T) {
	// Description
	// A valid issuer key signs an entry that verifies with its public key.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@example.com")

	// Act
	signErr := entry.Sign(entity, "")
	valid, verifyErr := entry.Verify(entity)

	// Assert
	require.NoError(t, signErr)
	require.NoError(t, verifyErr)
	assert.True(t, valid)
	assert.NotEmpty(t, entry.signature)
	assert.Equal(t, "proficiency@example.com", entry.signedBy)
}

func TestEntrySignAndVerifyAfterJSONRoundTrip(t *testing.T) {
	// Description
	// JSON storage preserves enough signed state for later verification.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	signErr := entry.Sign(entity, "")
	require.NoError(t, signErr)

	// Act
	data, marshalErr := entry.MarshalJSON()
	var decoded Entry
	unmarshalErr := decoded.UnmarshalJSON(data)
	valid, verifyErr := decoded.Verify(entity)

	// Assert
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	require.NoError(t, verifyErr)
	assert.True(t, valid)
	assert.Equal(t, entry.signature, decoded.signature)
	assert.Equal(t, entry.signedBy, decoded.signedBy)
}

func TestEntrySignAndVerifyAfterYAMLRoundTrip(t *testing.T) {
	// Description
	// YAML storage preserves detached signature armor and signer metadata.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	signErr := entry.Sign(entity, "")
	require.NoError(t, signErr)

	// Act
	data, marshalErr := entry.MarshalYAML()
	var decoded Entry
	unmarshalErr := decoded.UnmarshalYAML(data)
	valid, verifyErr := decoded.Verify(entity)

	// Assert
	require.NoError(t, marshalErr)
	require.NoError(t, unmarshalErr)
	require.NoError(t, verifyErr)
	assert.True(t, valid)
	assert.Equal(t, entry.signature, decoded.signature)
}

func TestEntrySignRelocksEncryptedPrivateKey(t *testing.T) {
	// Description
	// Signing leaves caller-owned encrypted private key material locked.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	passphrase := "correct horse battery staple"
	encryptErr := entity.EncryptPrivateKeys([]byte(passphrase), nil)
	require.NoError(t, encryptErr)

	// Act
	err := entry.Sign(entity, passphrase)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, entry.signature)
	assert.True(t, entity.PrivateKey.Encrypted)
}

func TestEntrySignRejectsWrongPassphrase(t *testing.T) {
	// Description
	// An encrypted private key remains unusable with the wrong passphrase.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	encryptErr := entity.EncryptPrivateKeys([]byte("correct passphrase"), nil)
	require.NoError(t, encryptErr)

	// Act
	err := entry.Sign(entity, "wrong passphrase")

	// Assert
	require.Error(t, err)
	assert.Empty(t, entry.signature)
	assert.Empty(t, entry.signedBy)
}

func TestEntrySignChecksPassphraseAfterSuccessfulSign(t *testing.T) {
	// Description
	// A prior successful signature does not leave the key unlocked for a later attempt.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	passphrase := "correct horse battery staple"
	encryptErr := entity.EncryptPrivateKeys([]byte(passphrase), nil)
	require.NoError(t, encryptErr)
	firstSignErr := entry.Sign(entity, passphrase)
	require.NoError(t, firstSignErr)
	originalSignature := entry.signature

	// Act
	err := entry.Sign(entity, "wrong passphrase")

	// Assert
	require.Error(t, err)
	assert.Equal(t, originalSignature, entry.signature)
	assert.True(t, entity.PrivateKey.Encrypted)
}

func TestEntrySignValidatesEntryBeforeSigning(t *testing.T) {
	// Description
	// Invalid protected fields cannot be signed.

	// Arrange
	entry := validTestEntry()
	entry.Topic = "Invalid Topic"
	entity := newTranscriptTestEntity(t, "proficiency@example.com")

	// Act
	err := entry.Sign(entity, "")

	// Assert
	require.Error(t, err)
	assert.ErrorContains(t, err, "validate")
	assert.Empty(t, entry.signature)
}

func TestEntrySignRejectsSignerOutsideIssuerDomain(t *testing.T) {
	// Description
	// Issuers cannot sign entries with a key from another domain.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@other.example")

	// Act
	err := entry.Sign(entity, "")

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrIssuerMismatch))
	assert.Empty(t, entry.signature)
}

func TestEntryVerifyRejectsUnsignedEntry(t *testing.T) {
	// Description
	// Verification fails closed when no detached signature is stored.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@example.com")

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
}

func TestEntryVerifyRejectsWrongPublicKey(t *testing.T) {
	// Description
	// A signature cannot be verified with an unrelated public key.

	// Arrange
	entry := validTestEntry()
	signer := newTranscriptTestEntity(t, "proficiency@example.com")
	wrongKey := newTranscriptTestEntity(t, "alternate@example.com")
	signErr := entry.Sign(signer, "")
	require.NoError(t, signErr)

	// Act
	valid, err := entry.Verify(wrongKey)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
}

func TestEntryVerifyRejectsTamperedSignedBy(t *testing.T) {
	// Description
	// Convenience signer metadata must agree with the verified key identity.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	signErr := entry.Sign(entity, "")
	require.NoError(t, signErr)
	entry.signedBy = "alternate@example.com"

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrIssuerMismatch))
}

func TestEntryVerifyRejectsCryptographicallyValidIssuerMismatch(t *testing.T) {
	// Description
	// A valid signature is insufficient when the key email domain differs from IssuedBy.

	// Arrange
	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@other.example")
	signEntryWithoutIssuerCheck(t, &entry, entity)

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrIssuerMismatch))
}

func TestEntryProtectedUserEmailBecomesStale(t *testing.T) {
	// Description
	// Changing the user identifier invalidates local signature freshness.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.UserEmail = "another@example.com"

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrSignatureStale))
}

func TestEntryProtectedTopicBecomesStale(t *testing.T) {
	// Description
	// Changing the topic invalidates local signature freshness.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.Topic = "subtraction"

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrSignatureStale))
}

func TestEntryProtectedTopicListBecomesStale(t *testing.T) {
	// Description
	// Changing the topic-list name invalidates local signature freshness.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.TopicList = "advanced-math"

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrSignatureStale))
}

func TestEntryProtectedTopicListVersionBecomesStale(t *testing.T) {
	// Description
	// Changing the topic-list version invalidates local signature freshness.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.TopicListVersion = "0.2.0"

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrSignatureStale))
}

func TestEntryProtectedTopicListOwnerBecomesStale(t *testing.T) {
	// Description
	// Changing the topic-list owner invalidates local signature freshness.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.TopicListOwner = "standards.example"

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrSignatureStale))
}

func TestEntryProtectedScoreBecomesStale(t *testing.T) {
	// Description
	// Changing the proficiency score invalidates local signature freshness.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.Score = score.Fluent

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrSignatureStale))
}

func TestEntryProtectedIssuedAtBecomesStale(t *testing.T) {
	// Description
	// Changing the issuance timestamp invalidates local signature freshness.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.IssuedAt = entry.IssuedAt.AddDate(0, 0, 1)

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrSignatureStale))
}

func TestEntryProtectedValidUntilBecomesStale(t *testing.T) {
	// Description
	// Changing the expiration timestamp invalidates local signature freshness.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.ValidUntil = entry.ValidUntil.AddDate(1, 0, 0)

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrSignatureStale))
}

func TestEntryProtectedIssuedByBecomesStale(t *testing.T) {
	// Description
	// Changing the issuer invalidates local signature freshness.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.IssuedBy = "other.example"

	// Act
	valid, err := entry.Verify(entity)

	// Assert
	require.Error(t, err)
	assert.False(t, valid)
	assert.True(t, errors.Is(err, ErrSignatureStale))
}

func TestEntryStaleJSONEncodingUsesNullSignatureMetadata(t *testing.T) {
	// Description
	// Stale protected content remains storable but does not expose an obsolete signature.

	// Arrange
	entry, _ := signedTestEntry(t)
	entry.Topic = "subtraction"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.NoError(t, err)
	assert.Contains(t, string(data), `"signature":null`)
	assert.Contains(t, string(data), `"signed-by":null`)
	assert.NotContains(t, string(data), "BEGIN PGP SIGNATURE")
}

func TestEntryStaleYAMLEncodingUsesNullSignatureMetadata(t *testing.T) {
	// Description
	// YAML also replaces stale signature metadata with null values.

	// Arrange
	entry, _ := signedTestEntry(t)
	entry.Score = score.Fluent

	// Act
	data, err := entry.MarshalYAML()

	// Assert
	require.NoError(t, err)
	assert.Contains(t, string(data), "signature: null\n")
	assert.Contains(t, string(data), "signed-by: null\n")
	assert.NotContains(t, string(data), "BEGIN PGP SIGNATURE")
}

func TestEntryTopicListSourcesRemainMutableAfterSigning(t *testing.T) {
	// Description
	// Redistribution pointers are excluded from protected signature content.

	// Arrange
	entry, entity := signedTestEntry(t)
	entry.TopicListSources = []string{
		"https://example.com/topic-lists/math/0.1.0",
		"https://mirror.example/math/0.1.0",
	}

	// Act
	valid, verifyErr := entry.Verify(entity)
	data, marshalErr := entry.MarshalJSON()

	// Assert
	require.NoError(t, verifyErr)
	require.NoError(t, marshalErr)
	assert.True(t, valid)
	assert.Contains(t, string(data), "mirror.example")
	assert.Contains(t, string(data), "BEGIN PGP SIGNATURE")
}

func TestEntryProtectedVerificationURLBecomesStale(t *testing.T) {
	// Description
	// The undocumented mutability of the verification location uses the secure protected default.

	// Arrange
	entry := validTestEntry()
	entry.VerificationURL = "https://example.com/verify"
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	signErr := entry.Sign(entity, "")
	require.NoError(t, signErr)
	entry.VerificationURL = "https://verify.example/status"

	// Act
	valid, verifyErr := entry.Verify(entity)
	data, marshalErr := entry.MarshalJSON()

	// Assert
	require.Error(t, verifyErr)
	require.NoError(t, marshalErr)
	assert.False(t, valid)
	assert.True(t, errors.Is(verifyErr, ErrSignatureStale))
	assert.Contains(t, string(data), `"verification-url":"https://verify.example/status"`)
	assert.Contains(t, string(data), `"signature":null`)
	assert.Contains(t, string(data), `"signed-by":null`)
}

func TestEntrySignedEncodingRejectsMismatchedStoredSigner(t *testing.T) {
	// Description
	// Encoding cannot emit signed metadata from a domain other than the issuer.

	// Arrange
	entry, _ := signedTestEntry(t)
	entry.signedBy = "proficiency@other.example"

	// Act
	data, err := entry.MarshalJSON()

	// Assert
	require.Error(t, err)
	assert.Nil(t, data)
	assert.True(t, errors.Is(err, ErrIssuerMismatch))
}

func signedTestEntry(t *testing.T) (Entry, *openpgp.Entity) {
	t.Helper()

	entry := validTestEntry()
	entity := newTranscriptTestEntity(t, "proficiency@example.com")
	err := entry.Sign(entity, "")
	require.NoError(t, err)

	return entry, entity
}

func newTranscriptTestEntity(t *testing.T, email string) *openpgp.Entity {
	t.Helper()

	entity, err := openpgp.NewEntity("OPM Transcript Test", "", email, &packet.Config{
		Algorithm: packet.PubKeyAlgoEdDSA,
	})
	require.NoError(t, err)

	return entity
}

func signEntryWithoutIssuerCheck(t *testing.T, entry *Entry, entity *openpgp.Entity) {
	t.Helper()

	protected := entry.protected()
	message, canonicalErr := canonical.JSON(protected)
	require.NoError(t, canonicalErr)
	signature, signedBy, signErr := pgp.SignWithPassphrase(entity, "", message)
	require.NoError(t, signErr)
	state, stateErr := canonical.NewState(protected)
	require.NoError(t, stateErr)

	entry.signature = signature
	entry.signedBy = signedBy
	entry.protectedState = state

	unsigned := *entry
	unsigned.signature = ""
	unsigned.signedBy = ""
	unsigned.protectedState = canonical.State{}
	wire, _, wireErr := unsigned.toWire()
	require.NoError(t, wireErr)
	wire.Signature = &signature
	wire.SignedBy = &signedBy
	data, marshalErr := json.Marshal(wire)
	require.NoError(t, marshalErr)

	var decoded Entry
	unmarshalErr := decoded.UnmarshalJSON(data)
	require.NoError(t, unmarshalErr)
	*entry = decoded
}
