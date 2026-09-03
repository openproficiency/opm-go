package transcript

import (
	"fmt"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/openproficiency/opm-go/internal/canonical"
	"github.com/openproficiency/opm-go/internal/pgp"
)

// Sign validates and signs the protected fields of an entry.
func (entry *Entry) Sign(privateKey *openpgp.Entity, passphrase string) error {
	if entry == nil {
		return fmt.Errorf("sign transcript entry: entry is nil")
	}
	if err := entry.validate(); err != nil {
		return fmt.Errorf("sign transcript entry: %w", err)
	}

	signerEmail, err := pgp.SignerEmail(privateKey)
	if err != nil {
		return fmt.Errorf("sign transcript entry: %w", err)
	}
	if !emailDomainMatches(signerEmail, entry.IssuedBy) {
		return fmt.Errorf("%w: signed by %q, issued by %q", ErrIssuerMismatch, signerEmail, entry.IssuedBy)
	}
	protected := entry.protected()
	message, err := canonical.JSON(protected)
	if err != nil {
		return fmt.Errorf("sign transcript entry: %w", err)
	}
	signature, signedBy, err := pgp.SignWithPassphrase(privateKey, passphrase, message)
	if err != nil {
		return fmt.Errorf("sign transcript entry: %w", err)
	}
	if !emailDomainMatches(signedBy, entry.IssuedBy) {
		return fmt.Errorf("%w: signed by %q, issued by %q", ErrIssuerMismatch, signedBy, entry.IssuedBy)
	}
	protectedState, err := canonical.NewState(protected)
	if err != nil {
		return fmt.Errorf("sign transcript entry: %w", err)
	}

	entry.signature = signature
	entry.signedBy = signedBy
	entry.protectedState = protectedState
	return nil
}

// Verify reports whether an entry has a fresh signature made by the issuer.
func (entry Entry) Verify(publicKey *openpgp.Entity) (bool, error) {
	if entry.signature == "" || entry.signedBy == "" {
		return false, errSignatureMissing
	}

	matches, err := entry.protectedState.Matches(entry.protected())
	if err != nil {
		return false, fmt.Errorf("verify transcript entry: compare protected state: %w", err)
	}
	if !matches {
		return false, ErrSignatureStale
	}

	message, err := canonical.JSON(entry.protected())
	if err != nil {
		return false, fmt.Errorf("verify transcript entry: %w", err)
	}
	verifiedBy, err := pgp.Verify(publicKey, message, entry.signature)
	if err != nil {
		return false, fmt.Errorf("verify transcript entry: %w", err)
	}
	if verifiedBy != entry.signedBy {
		return false, fmt.Errorf("%w: signature identity %q differs from signed-by %q", ErrIssuerMismatch, verifiedBy, entry.signedBy)
	}
	if !emailDomainMatches(verifiedBy, entry.IssuedBy) {
		return false, fmt.Errorf("%w: signed by %q, issued by %q", ErrIssuerMismatch, verifiedBy, entry.IssuedBy)
	}

	return true, nil
}
