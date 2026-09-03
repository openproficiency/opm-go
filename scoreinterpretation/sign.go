package scoreinterpretation

import (
	"errors"
	"fmt"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/openproficiency/opm-go/internal/canonical"
	"github.com/openproficiency/opm-go/internal/pgp"
)

// Sign validates and signs the protected fields of the list.
func (list *List) Sign(privateKey *openpgp.Entity, passphrase string) error {
	if list == nil {
		return errors.New("sign score interpretation list: nil receiver")
	}
	if err := list.Validate(); err != nil {
		return err
	}

	signerEmail, err := pgp.SignerEmail(privateKey)
	if err != nil {
		return fmt.Errorf("sign score interpretation list: %w", err)
	}
	if err := signerMatchesOwner(signerEmail, list.Owner); err != nil {
		return fmt.Errorf("sign score interpretation list: %w", err)
	}

	protected, err := list.protectedDocument()
	if err != nil {
		return fmt.Errorf("sign score interpretation list: %w", err)
	}
	message, err := canonical.JSON(protected)
	if err != nil {
		return fmt.Errorf("sign score interpretation list: %w", err)
	}
	signature, signedBy, err := pgp.SignWithPassphrase(privateKey, passphrase, message)
	if err != nil {
		return fmt.Errorf("sign score interpretation list: %w", err)
	}
	if err := signerMatchesOwner(signedBy, list.Owner); err != nil {
		return fmt.Errorf("sign score interpretation list: %w", err)
	}
	state, err := canonical.NewState(protected)
	if err != nil {
		return fmt.Errorf("sign score interpretation list: %w", err)
	}

	list.signature = &signature
	list.signedBy = &signedBy
	list.signatureState = state
	return nil
}

// Signature returns the current ASCII-armored detached signature.
func (list List) Signature() (string, error) {
	if err := list.checkSignature(); err != nil {
		return "", err
	}
	return *list.signature, nil
}

// SignedBy returns the signer email stored with the current signature.
func (list List) SignedBy() (string, error) {
	if err := list.checkSignature(); err != nil {
		return "", err
	}
	return *list.signedBy, nil
}

// SignatureKeyID returns the issuer key ID embedded in the current signature.
func (list List) SignatureKeyID() (uint64, error) {
	if err := list.checkSignature(); err != nil {
		return 0, err
	}
	return pgp.SignatureKeyID(*list.signature)
}

// Verify verifies the current signature and requires the signer domain to match Owner.
func (list List) Verify(publicKey *openpgp.Entity) (bool, error) {
	if err := list.checkSignature(); err != nil {
		return false, err
	}

	protected, err := list.protectedDocument()
	if err != nil {
		return false, fmt.Errorf("verify score interpretation list: %w", err)
	}
	message, err := canonical.JSON(protected)
	if err != nil {
		return false, fmt.Errorf("verify score interpretation list: %w", err)
	}
	verifiedBy, err := pgp.Verify(publicKey, message, *list.signature)
	if err != nil {
		return false, fmt.Errorf("verify score interpretation list: %w", err)
	}
	if err := signerMatchesOwner(verifiedBy, list.Owner); err != nil {
		return false, fmt.Errorf("verify score interpretation list: %w", err)
	}
	if verifiedBy != *list.signedBy {
		return false, fmt.Errorf(
			"verify score interpretation list: signature identity %q differs from signed-by %q",
			verifiedBy,
			*list.signedBy,
		)
	}

	return true, nil
}

func (list List) checkSignature() error {
	if list.signature == nil || list.signedBy == nil || !list.signatureState.Initialized() {
		return errUnsigned
	}

	protected, err := list.protectedDocument()
	if err != nil {
		return fmt.Errorf("check score interpretation list signature state: %w", err)
	}
	matches, err := list.signatureState.Matches(protected)
	if err != nil {
		return fmt.Errorf("check score interpretation list signature state: %w", err)
	}
	if !matches {
		return ErrSignatureStale
	}
	return nil
}
