package transcript

import "errors"

var (
	// ErrSignatureStale indicates that protected fields changed after signing.
	ErrSignatureStale = errors.New("transcript entry signature is stale")

	// ErrIssuerMismatch indicates that the signing email domain does not match IssuedBy.
	ErrIssuerMismatch = errors.New("transcript entry signer does not match issuer")
)

var errSignatureMissing = errors.New("transcript entry signature is missing")
