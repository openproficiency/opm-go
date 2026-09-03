// Package pgp signs and verifies protected OPM content with OpenPGP.
package pgp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

var signMutex sync.Mutex

// Decrypt unlocks all encrypted private keys on an entity.
func Decrypt(entity *openpgp.Entity, passphrase string) error {
	if entity == nil {
		return errors.New("decrypt OpenPGP entity: entity is nil")
	}
	if !hasPrivateKey(entity) {
		return errors.New("decrypt OpenPGP entity: private key is missing")
	}

	if err := entity.DecryptPrivateKeys([]byte(passphrase)); err != nil {
		return fmt.Errorf("decrypt OpenPGP private keys: %w", err)
	}

	return nil
}

func hasPrivateKey(entity *openpgp.Entity) bool {
	if entity.PrivateKey != nil {
		return true
	}

	for _, subkey := range entity.Subkeys {
		if subkey.PrivateKey != nil {
			return true
		}
	}

	return false
}

func encryptedPrivateKeys(entity *openpgp.Entity) []*packet.PrivateKey {
	if entity == nil {
		return nil
	}

	var privateKeys []*packet.PrivateKey
	if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
		privateKeys = append(privateKeys, entity.PrivateKey)
	}
	for _, subkey := range entity.Subkeys {
		if subkey.PrivateKey != nil && subkey.PrivateKey.Encrypted {
			privateKeys = append(privateKeys, subkey.PrivateKey)
		}
	}

	return privateKeys
}

// Sign creates an ASCII-armored detached signature and returns its signer email.
func Sign(entity *openpgp.Entity, message []byte) (string, string, error) {
	if entity == nil {
		return "", "", errors.New("sign content: OpenPGP entity is nil")
	}

	signerEmail, err := SignerEmail(entity)
	if err != nil {
		return "", "", err
	}

	var signature strings.Builder
	if err := openpgp.ArmoredDetachSign(&signature, entity, bytes.NewReader(message), nil); err != nil {
		return "", "", fmt.Errorf("sign content: %w", err)
	}

	return signature.String(), signerEmail, nil
}

// SignWithPassphrase signs without leaving encrypted caller keys unlocked.
func SignWithPassphrase(
	entity *openpgp.Entity,
	passphrase string,
	message []byte,
) (string, string, error) {
	signMutex.Lock()
	defer signMutex.Unlock()

	encryptedKeys := encryptedPrivateKeys(entity)
	if err := Decrypt(entity, passphrase); err != nil {
		return "", "", err
	}

	signature, signerEmail, signErr := Sign(entity, message)
	relockErr := packet.EncryptPrivateKeys(encryptedKeys, []byte(passphrase), nil)
	if signErr != nil {
		if relockErr != nil {
			return "", "", errors.Join(signErr, fmt.Errorf("re-encrypt OpenPGP private keys: %w", relockErr))
		}
		return "", "", signErr
	}
	if relockErr != nil {
		return "", "", fmt.Errorf("re-encrypt OpenPGP private keys: %w", relockErr)
	}

	return signature, signerEmail, nil
}

// SignatureKeyID extracts the issuer key ID from an armored signature.
func SignatureKeyID(armoredSignature string) (uint64, error) {
	block, err := armor.Decode(strings.NewReader(armoredSignature))
	if err != nil {
		return 0, fmt.Errorf("decode armored signature: %w", err)
	}
	if block.Type != openpgp.SignatureType {
		return 0, fmt.Errorf("decode armored signature: unexpected armor type %q", block.Type)
	}

	reader := packet.NewReader(block.Body)
	for {
		parsedPacket, nextErr := reader.Next()
		if errors.Is(nextErr, io.EOF) {
			return 0, errors.New("read armored signature: signature packet is missing")
		}
		if nextErr != nil {
			return 0, fmt.Errorf("read armored signature: %w", nextErr)
		}

		signature, ok := parsedPacket.(*packet.Signature)
		if !ok {
			continue
		}
		if signature.IssuerKeyId != nil {
			return *signature.IssuerKeyId, nil
		}
		if len(signature.IssuerFingerprint) >= 8 {
			return binary.BigEndian.Uint64(signature.IssuerFingerprint[len(signature.IssuerFingerprint)-8:]), nil
		}

		return 0, errors.New("read armored signature: issuer key ID is missing")
	}
}

// SignerEmail selects the primary non-revoked email from an entity.
func SignerEmail(entity *openpgp.Entity) (string, error) {
	if entity == nil {
		return "", errors.New("select signer email: OpenPGP entity is nil")
	}

	identityNames := make([]string, 0, len(entity.Identities))
	for identityName := range entity.Identities {
		identityNames = append(identityNames, identityName)
	}
	sort.Strings(identityNames)

	now := time.Now()
	for _, primaryOnly := range []bool{true, false} {
		for _, identityName := range identityNames {
			identity := entity.Identities[identityName]
			if identity == nil || identity.UserId == nil || identity.UserId.Email == "" || identity.Revoked(now) {
				continue
			}

			isPrimary := identity.SelfSignature != nil &&
				identity.SelfSignature.IsPrimaryId != nil &&
				*identity.SelfSignature.IsPrimaryId
			if isPrimary == primaryOnly {
				return identity.UserId.Email, nil
			}
		}
	}

	return "", errors.New("select signer email: entity has no usable email identity")
}

// Verify checks an armored detached signature and returns the verified signer email.
func Verify(entity *openpgp.Entity, message []byte, armoredSignature string) (string, error) {
	if entity == nil {
		return "", errors.New("verify signature: OpenPGP entity is nil")
	}

	signer, err := openpgp.CheckArmoredDetachedSignature(
		openpgp.EntityList{entity},
		bytes.NewReader(message),
		strings.NewReader(armoredSignature),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("verify signature: %w", err)
	}

	signerEmail, err := SignerEmail(signer)
	if err != nil {
		return "", fmt.Errorf("verify signature: %w", err)
	}

	return signerEmail, nil
}
