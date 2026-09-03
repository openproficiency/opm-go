// Package canonical produces deterministic JSON and tracks protected values.
package canonical

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// State records the SHA-256 digest of canonical protected content.
type State struct {
	digest      [sha256.Size]byte
	initialized bool
}

// JSON encodes a value as compact deterministic JSON.
func JSON(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("encode canonical JSON: %w", err)
	}

	return bytes.TrimSuffix(output.Bytes(), []byte{'\n'}), nil
}

// NewState captures the current digest of protected content.
func NewState(value any) (State, error) {
	digest, err := SHA256(value)
	if err != nil {
		return State{}, err
	}

	return State{
		digest:      digest,
		initialized: true,
	}, nil
}

// SHA256 returns the digest of canonical JSON.
func SHA256(value any) ([sha256.Size]byte, error) {
	canonicalJSON, err := JSON(value)
	if err != nil {
		return [sha256.Size]byte{}, err
	}

	return sha256.Sum256(canonicalJSON), nil
}

// Initialized reports whether protected content has been captured.
func (state State) Initialized() bool {
	return state.initialized
}

// Matches reports whether protected content still has the captured digest.
func (state State) Matches(value any) (bool, error) {
	if !state.initialized {
		return false, nil
	}

	digest, err := SHA256(value)
	if err != nil {
		return false, err
	}

	return digest == state.digest, nil
}
