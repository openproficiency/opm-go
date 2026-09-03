package scoreinterpretation

import (
	"time"

	"github.com/openproficiency/opm-go/internal/canonical"
	"github.com/openproficiency/opm-go/topic"
)

// List groups related score interpretations for distribution.
type List struct {
	Owner           string
	Name            string
	Description     string
	Version         string
	IssuedAt        time.Time
	Interpretations map[string]Interpretation
	Dependencies    map[string]topic.Dependency

	schemaURL      string
	signature      *string
	signedBy       *string
	signatureState canonical.State

	dependencyLongForm         map[string]bool
	dependencyLocationsPresent map[string]bool
}

// Add inserts or replaces an interpretation using its ID as the map key.
func (list *List) Add(interpretation Interpretation) {
	if list.Interpretations == nil {
		list.Interpretations = make(map[string]Interpretation)
	}
	list.Interpretations[interpretation.ID] = interpretation
}
