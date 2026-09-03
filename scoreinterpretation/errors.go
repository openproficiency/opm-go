package scoreinterpretation

import "errors"

var (
	// ErrSignatureStale reports that protected list content changed after signing.
	ErrSignatureStale = errors.New("score interpretation list signature is stale")

	errUnsigned = errors.New("score interpretation list is unsigned")
)
