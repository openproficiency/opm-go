package scoreinterpretation

// Interpretation defines a named outcome and the requirements needed to earn it.
type Interpretation struct {
	ID           string
	Name         string
	Description  string
	Requirements []Requirement

	schemaURL string
}
