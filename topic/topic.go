// Package topic defines Open Proficiency Model topic lists.
package topic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/openproficiency/opm-go/internal/canonical"
	"github.com/openproficiency/opm-go/internal/pgp"
	"github.com/openproficiency/opm-go/internal/schema"
	"github.com/openproficiency/opm-go/internal/semantic"
	"gopkg.in/yaml.v3"
)

const maxTopicDepth = 100

var (
	// ErrSignatureStale reports that protected list content changed after signing.
	ErrSignatureStale = errors.New("topic list signature is stale")

	errUnsigned = errors.New("topic list is unsigned")
)

// Topic is a unique area of knowledge within a topic list.
type Topic struct {
	ID             string
	DisplayName    string
	Description    string
	DocsURL        string
	ValidityPeriod int
	Subtopics      []string
	Pretopics      []string

	schemaURL       string
	inlineSubtopics map[int]Topic
}

// Dependency identifies an external topic list.
type Dependency struct {
	Owner     string
	Name      string
	Version   string
	Locations []string

	longForm         bool
	locationsPresent bool
}

// List is an OPM topic list.
type List struct {
	Owner        string
	Name         string
	Description  string
	Version      string
	IssuedAt     time.Time
	Topics       map[string]Topic
	Dependencies map[string]Dependency

	schemaURL      string
	signature      *string
	signedBy       *string
	signatureState canonical.State
}

// Report summarizes the size and local subtopic complexity of a list.
type Report struct {
	TopicCount        int
	AtomicTopicCount  int
	GroupTopicCount   int
	MaxSubtopics      int
	MaxDepth          int
	DependenciesCount int
}

// Add inserts or replaces a topic using its ID as the map key.
func (list *List) Add(topic Topic) {
	if list.Topics == nil {
		list.Topics = make(map[string]Topic)
	}
	list.Topics[topic.ID] = topic
}

// FullName returns owner/name@version.
func (list List) FullName() string {
	return fmt.Sprintf("%s/%s@%s", list.Owner, list.Name, list.Version)
}

// AtomicTopics returns a new map containing topics without subtopics.
func (list List) AtomicTopics() map[string]Topic {
	allTopics := list.topicsForReport()
	atomic := make(map[string]Topic)
	for id, topic := range allTopics {
		if len(topic.Subtopics) == 0 {
			atomic[id] = topic
		}
	}

	return atomic
}

// GroupTopics returns a new map containing topics with subtopics.
func (list List) GroupTopics() map[string]Topic {
	allTopics := list.topicsForReport()
	groups := make(map[string]Topic)
	for id, topic := range allTopics {
		if len(topic.Subtopics) != 0 {
			groups[id] = topic
		}
	}

	return groups
}

// ComplexityReport returns list-scope topic and dependency metrics.
func (list List) ComplexityReport() Report {
	allTopics := list.topicsForReport()
	report := Report{
		TopicCount:        len(allTopics),
		DependenciesCount: len(list.Dependencies),
	}

	for _, topic := range allTopics {
		if len(topic.Subtopics) == 0 {
			report.AtomicTopicCount++
		} else {
			report.GroupTopicCount++
		}
		if len(topic.Subtopics) > report.MaxSubtopics {
			report.MaxSubtopics = len(topic.Subtopics)
		}
	}

	for id := range allTopics {
		depth := topicDepth(id, allTopics, make(map[string]bool))
		if depth > report.MaxDepth {
			report.MaxDepth = depth
		}
	}

	return report
}

// MarshalJSON returns deterministic OPM-compatible JSON.
func (list List) MarshalJSON() ([]byte, error) {
	if err := list.Validate(); err != nil {
		return nil, err
	}

	document, err := list.wireDocument(true)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("marshal topic list JSON: %w", err)
	}

	return data, nil
}

// UnmarshalJSON loads an OPM-compatible JSON topic list.
func (list *List) UnmarshalJSON(data []byte) error {
	if list == nil {
		return errors.New("unmarshal topic list JSON: nil receiver")
	}
	if err := schema.ValidateJSON(schema.TopicList, data); err != nil {
		return err
	}

	decoded, err := decodeJSON(data)
	if err != nil {
		return err
	}
	if err := decoded.Validate(); err != nil {
		return err
	}
	if decoded.signature != nil {
		state, stateErr := canonical.NewState(decoded.protectedDocument())
		if stateErr != nil {
			return fmt.Errorf("capture topic list signature state: %w", stateErr)
		}
		decoded.signatureState = state
	}

	*list = decoded
	return nil
}

// MarshalYAML returns deterministic OPM-compatible YAML.
func (list List) MarshalYAML() ([]byte, error) {
	if err := list.Validate(); err != nil {
		return nil, err
	}

	document, err := list.wireDocument(true)
	if err != nil {
		return nil, err
	}

	data, err := yaml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("marshal topic list YAML: %w", err)
	}

	return data, nil
}

// UnmarshalYAML loads an OPM-compatible YAML topic list.
func (list *List) UnmarshalYAML(data []byte) error {
	if list == nil {
		return errors.New("unmarshal topic list YAML: nil receiver")
	}
	if err := schema.ValidateYAML(schema.TopicList, data); err != nil {
		return err
	}

	var document any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("unmarshal topic list YAML: %w", err)
	}
	jsonData, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("normalize topic list YAML: %w", err)
	}

	decoded, err := decodeJSON(jsonData)
	if err != nil {
		return err
	}
	if err := decoded.Validate(); err != nil {
		return err
	}
	if decoded.signature != nil {
		state, stateErr := canonical.NewState(decoded.protectedDocument())
		if stateErr != nil {
			return fmt.Errorf("capture topic list signature state: %w", stateErr)
		}
		decoded.signatureState = state
	}

	*list = decoded
	return nil
}

// Validate checks OPM schema constraints and local topic graph integrity.
func (list List) Validate() error {
	if list.IssuedAt.IsZero() {
		return errors.New("validate topic list: issued-at timestamp is required")
	}
	if (list.signature == nil) != (list.signedBy == nil) {
		return errors.New("validate topic list: signature and signed-by must both be null or both be set")
	}
	if list.signedBy != nil {
		if err := signerMatchesOwner(*list.signedBy, list.Owner); err != nil {
			return fmt.Errorf("validate topic list: %w", err)
		}
	}

	document, err := list.wireDocument(true)
	if err != nil {
		return err
	}
	data, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("validate topic list: marshal JSON: %w", err)
	}
	if err := schema.ValidateJSON(schema.TopicList, data); err != nil {
		return err
	}

	allTopics, err := list.allTopics()
	if err != nil {
		return err
	}
	if err := validateTopicReferences(allTopics, list.Dependencies); err != nil {
		return err
	}
	if err := validateSubtopicGraph(allTopics); err != nil {
		return err
	}
	if err := validatePretopicGraph(allTopics); err != nil {
		return err
	}

	for id := range allTopics {
		if depth := topicDepth(id, allTopics, make(map[string]bool)); depth > maxTopicDepth {
			return fmt.Errorf("validate topic list: topic %q exceeds maximum subtopic depth %d", id, maxTopicDepth)
		}
	}

	return nil
}

// Sign signs the protected list fields with an OpenPGP private key.
func (list *List) Sign(privateKey *openpgp.Entity, passphrase string) error {
	if list == nil {
		return errors.New("sign topic list: nil receiver")
	}
	if err := list.Validate(); err != nil {
		return err
	}

	signerEmail, err := pgp.SignerEmail(privateKey)
	if err != nil {
		return err
	}
	if err := signerMatchesOwner(signerEmail, list.Owner); err != nil {
		return fmt.Errorf("sign topic list: %w", err)
	}
	protected := list.protectedDocument()
	message, err := canonical.JSON(protected)
	if err != nil {
		return fmt.Errorf("encode protected topic list: %w", err)
	}
	signature, signedBy, err := pgp.SignWithPassphrase(privateKey, passphrase, message)
	if err != nil {
		return err
	}
	if err := signerMatchesOwner(signedBy, list.Owner); err != nil {
		return fmt.Errorf("sign topic list: %w", err)
	}

	state, err := canonical.NewState(protected)
	if err != nil {
		return fmt.Errorf("capture topic list signature state: %w", err)
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

	message, err := canonical.JSON(list.protectedDocument())
	if err != nil {
		return false, fmt.Errorf("encode protected topic list: %w", err)
	}
	verifiedBy, err := pgp.Verify(publicKey, message, *list.signature)
	if err != nil {
		return false, err
	}
	if err := signerMatchesOwner(verifiedBy, list.Owner); err != nil {
		return false, fmt.Errorf("verify topic list: %w", err)
	}
	if verifiedBy != *list.signedBy {
		return false, fmt.Errorf(
			"verify topic list: signature identity %q differs from signed-by %q",
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

	matches, err := list.signatureState.Matches(list.protectedDocument())
	if err != nil {
		return fmt.Errorf("check topic list signature state: %w", err)
	}
	if !matches {
		return ErrSignatureStale
	}

	return nil
}

func (list List) wireDocument(includeSignature bool) (wireList, error) {
	topics := make(map[string]wireTopic, len(list.Topics))
	for id, topic := range list.Topics {
		topics[id] = makeWireTopic(topic, topic.ID == id)
	}

	dependencies := make(map[string]any, len(list.Dependencies))
	for alias, dependency := range list.Dependencies {
		dependencies[alias] = dependencyWireValue(dependency, false)
	}

	var signature *string
	var signedBy *string
	if includeSignature && list.checkSignature() == nil {
		signature = list.signature
		signedBy = list.signedBy
	}

	return wireList{
		Schema:       list.schemaURL,
		Owner:        list.Owner,
		Name:         list.Name,
		Description:  list.Description,
		Version:      list.Version,
		IssuedAt:     list.IssuedAt.UTC(),
		Signature:    signature,
		SignedBy:     signedBy,
		Topics:       topics,
		Dependencies: dependencies,
	}, nil
}

func (list List) protectedDocument() protectedList {
	topics := make(map[string]protectedTopic, len(list.Topics))
	for id, topic := range list.Topics {
		topics[id] = makeProtectedTopic(topic, topic.ID == id)
	}

	dependencies := make(map[string]protectedDependency, len(list.Dependencies))
	for alias, dependency := range list.Dependencies {
		dependencies[alias] = protectedDependency{
			Owner:   dependency.Owner,
			Name:    dependency.Name,
			Version: dependency.Version,
		}
	}

	return protectedList{
		Owner:        list.Owner,
		Name:         list.Name,
		Description:  list.Description,
		Version:      list.Version,
		IssuedAt:     formatTime(list.IssuedAt),
		Topics:       topics,
		Dependencies: dependencies,
	}
}

func makeProtectedTopic(topic Topic, omitMatchingID bool) protectedTopic {
	id := topic.ID
	if omitMatchingID {
		id = ""
	}

	var subtopics []any
	if len(topic.Subtopics) != 0 {
		subtopics = make([]any, 0, len(topic.Subtopics))
		for index, id := range topic.Subtopics {
			if inline, exists := topic.inlineSubtopics[index]; exists && inline.ID == id {
				subtopics = append(subtopics, makeProtectedTopic(inline, false))
			} else {
				subtopics = append(subtopics, id)
			}
		}
	}

	return protectedTopic{
		ID:             id,
		DisplayName:    topic.DisplayName,
		Description:    topic.Description,
		DocsURL:        topic.DocsURL,
		ValidityPeriod: topic.ValidityPeriod,
		Subtopics:      subtopics,
		Pretopics:      topic.Pretopics,
	}
}

func (list List) allTopics() (map[string]Topic, error) {
	all := make(map[string]Topic)
	for key, topic := range list.Topics {
		if topic.ID == "" {
			return nil, fmt.Errorf("validate topic list: topic map key %q has an empty ID", key)
		}
		if topic.ID != key {
			return nil, fmt.Errorf("validate topic list: topic map key %q does not match ID %q", key, topic.ID)
		}
		if err := collectTopic(topic, all, make(map[string]bool), 1); err != nil {
			return nil, err
		}
	}

	return all, nil
}

func (list List) topicsForReport() map[string]Topic {
	allTopics, err := list.allTopics()
	if err == nil {
		return allTopics
	}

	topLevel := make(map[string]Topic, len(list.Topics))
	for id, topic := range list.Topics {
		topLevel[id] = topic
	}

	return topLevel
}

func collectTopic(topic Topic, all map[string]Topic, path map[string]bool, depth int) error {
	if depth > maxTopicDepth {
		return fmt.Errorf("validate topic list: inline topic %q exceeds maximum depth %d", topic.ID, maxTopicDepth)
	}
	if topic.ID == "" {
		return errors.New("validate topic list: inline subtopic ID is required")
	}
	if path[topic.ID] {
		return fmt.Errorf("validate topic list: inline subtopic cycle includes %q", topic.ID)
	}
	if existing, exists := all[topic.ID]; exists {
		if !topicsEqual(existing, topic) {
			return fmt.Errorf("validate topic list: topic %q has conflicting definitions", topic.ID)
		}
		return nil
	}
	all[topic.ID] = topic

	path[topic.ID] = true
	defer delete(path, topic.ID)
	for index, inline := range topic.inlineSubtopics {
		if index < 0 || index >= len(topic.Subtopics) || topic.Subtopics[index] != inline.ID {
			continue
		}
		if err := collectTopic(inline, all, path, depth+1); err != nil {
			return err
		}
	}

	return nil
}

func topicsEqual(first Topic, second Topic) bool {
	firstJSON, firstErr := json.Marshal(makeWireTopic(first, false))
	secondJSON, secondErr := json.Marshal(makeWireTopic(second, false))
	return firstErr == nil && secondErr == nil && bytes.Equal(firstJSON, secondJSON)
}

func validateTopicReferences(topics map[string]Topic, dependencies map[string]Dependency) error {
	for id, topic := range topics {
		if err := semantic.UniqueStrings(topic.Subtopics); err != nil {
			return fmt.Errorf("validate topic %q subtopics: %w", id, err)
		}
		for _, subtopic := range topic.Subtopics {
			if _, exists := topics[subtopic]; !exists {
				return fmt.Errorf("validate topic %q: unknown subtopic %q", id, subtopic)
			}
		}

		if err := semantic.UniqueStrings(topic.Pretopics); err != nil {
			return fmt.Errorf("validate topic %q pretopics: %w", id, err)
		}
		for _, pretopic := range topic.Pretopics {
			if !strings.Contains(pretopic, ".") {
				if _, exists := topics[pretopic]; !exists {
					return fmt.Errorf("validate topic %q: unknown pretopic %q", id, pretopic)
				}
				continue
			}

			alias, externalTopic, found := strings.Cut(pretopic, ".")
			if !found || strings.Contains(externalTopic, ".") {
				return fmt.Errorf("validate topic %q: invalid external pretopic %q", id, pretopic)
			}
			if err := semantic.KebabCase(alias); err != nil {
				return fmt.Errorf("validate topic %q external pretopic: %w", id, err)
			}
			if err := semantic.KebabCase(externalTopic); err != nil {
				return fmt.Errorf("validate topic %q external pretopic: %w", id, err)
			}
			if _, exists := dependencies[alias]; !exists {
				return fmt.Errorf("validate topic %q: unknown dependency alias %q", id, alias)
			}
		}
	}

	return nil
}

func validateSubtopicGraph(topics map[string]Topic) error {
	parentByChild := make(map[string]string)
	for parent, topic := range topics {
		for _, child := range topic.Subtopics {
			if previous, exists := parentByChild[child]; exists && previous != parent {
				return fmt.Errorf("validate topic list: subtopic %q is shared by groups %q and %q", child, previous, parent)
			}
			parentByChild[child] = parent
		}
	}

	state := make(map[string]uint8, len(topics))
	for id := range topics {
		if err := visitSubtopics(id, topics, state); err != nil {
			return err
		}
	}

	return nil
}

func visitSubtopics(id string, topics map[string]Topic, state map[string]uint8) error {
	if state[id] == 1 {
		return fmt.Errorf("validate topic list: subtopic cycle includes %q", id)
	}
	if state[id] == 2 {
		return nil
	}

	state[id] = 1
	for _, child := range topics[id].Subtopics {
		if err := visitSubtopics(child, topics, state); err != nil {
			return err
		}
	}
	state[id] = 2
	return nil
}

func validatePretopicGraph(topics map[string]Topic) error {
	state := make(map[string]uint8, len(topics))
	for id := range topics {
		if err := visitPretopics(id, topics, state); err != nil {
			return err
		}
	}

	return nil
}

func visitPretopics(id string, topics map[string]Topic, state map[string]uint8) error {
	if state[id] == 1 {
		return fmt.Errorf("validate topic list: pretopic cycle includes %q", id)
	}
	if state[id] == 2 {
		return nil
	}

	state[id] = 1
	for _, prerequisite := range topics[id].Pretopics {
		if strings.Contains(prerequisite, ".") {
			continue
		}
		if err := visitPretopics(prerequisite, topics, state); err != nil {
			return err
		}
	}
	state[id] = 2
	return nil
}

func topicDepth(id string, topics map[string]Topic, path map[string]bool) int {
	if path[id] {
		return 0
	}
	path[id] = true
	defer delete(path, id)

	topic, exists := topics[id]
	if !exists || len(topic.Subtopics) == 0 {
		return 1
	}

	maxChildDepth := 0
	for _, child := range topic.Subtopics {
		childDepth := topicDepth(child, topics, path)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}

	return 1 + maxChildDepth
}

func signerMatchesOwner(email string, owner string) error {
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return fmt.Errorf("invalid signer email %q", email)
	}

	at := strings.LastIndexByte(email, '@')
	if at < 0 || email[at+1:] != owner {
		return fmt.Errorf("signer email domain must exactly match owner %q", owner)
	}

	return nil
}

func makeWireTopic(topic Topic, omitMatchingID bool) wireTopic {
	id := topic.ID
	if omitMatchingID {
		id = ""
	}

	var subtopics []any
	if len(topic.Subtopics) != 0 {
		subtopics = make([]any, 0, len(topic.Subtopics))
		for index, id := range topic.Subtopics {
			if inline, exists := topic.inlineSubtopics[index]; exists && inline.ID == id {
				subtopics = append(subtopics, makeWireTopic(inline, false))
			} else {
				subtopics = append(subtopics, id)
			}
		}
	}

	return wireTopic{
		Schema:         topic.schemaURL,
		ID:             id,
		DisplayName:    topic.DisplayName,
		Description:    topic.Description,
		DocsURL:        topic.DocsURL,
		ValidityPeriod: topic.ValidityPeriod,
		Subtopics:      subtopics,
		Pretopics:      topic.Pretopics,
	}
}

func dependencyWireValue(dependency Dependency, protected bool) any {
	if !protected && !dependency.longForm && len(dependency.Locations) == 0 {
		return fmt.Sprintf("%s/%s@%s", dependency.Owner, dependency.Name, dependency.Version)
	}

	var locations *[]string
	if !protected && (dependency.locationsPresent || len(dependency.Locations) != 0) {
		copied := append([]string(nil), dependency.Locations...)
		locations = &copied
	}

	return wireDependency{
		Owner:     dependency.Owner,
		Name:      dependency.Name,
		Version:   dependency.Version,
		Locations: locations,
	}
}

func decodeJSON(data []byte) (List, error) {
	var raw rawList
	if err := json.Unmarshal(data, &raw); err != nil {
		return List{}, fmt.Errorf("unmarshal topic list JSON: %w", err)
	}

	decoded := List{
		Owner:        raw.Owner,
		Name:         raw.Name,
		Description:  raw.Description,
		Version:      raw.Version,
		IssuedAt:     raw.IssuedAt,
		Topics:       make(map[string]Topic, len(raw.Topics)),
		Dependencies: make(map[string]Dependency, len(raw.Dependencies)),
		schemaURL:    raw.Schema,
		signature:    raw.Signature,
		signedBy:     raw.SignedBy,
	}

	for id, rawTopic := range raw.Topics {
		topic, err := decodeTopicJSON(rawTopic, 1)
		if err != nil {
			return List{}, fmt.Errorf("decode topic %q: %w", id, err)
		}
		if topic.ID == "" {
			topic.ID = id
		} else if topic.ID != id {
			return List{}, fmt.Errorf("decode topic %q: ID %q does not match map key", id, topic.ID)
		}
		decoded.Topics[id] = topic
	}

	for alias, rawDependency := range raw.Dependencies {
		dependency, err := decodeDependencyJSON(rawDependency)
		if err != nil {
			return List{}, fmt.Errorf("decode dependency %q: %w", alias, err)
		}
		decoded.Dependencies[alias] = dependency
	}

	return decoded, nil
}

func decodeTopicJSON(data json.RawMessage, depth int) (Topic, error) {
	if depth > maxTopicDepth {
		return Topic{}, fmt.Errorf("maximum inline topic depth %d exceeded", maxTopicDepth)
	}

	var raw rawTopic
	if err := json.Unmarshal(data, &raw); err != nil {
		return Topic{}, err
	}

	topic := Topic{
		ID:             raw.ID,
		DisplayName:    raw.DisplayName,
		Description:    raw.Description,
		DocsURL:        raw.DocsURL,
		ValidityPeriod: raw.ValidityPeriod,
		Pretopics:      raw.Pretopics,
		schemaURL:      raw.Schema,
	}

	if len(raw.Subtopics) != 0 {
		topic.Subtopics = make([]string, 0, len(raw.Subtopics))
	}
	for index, item := range raw.Subtopics {
		var id string
		if err := json.Unmarshal(item, &id); err == nil {
			topic.Subtopics = append(topic.Subtopics, id)
			continue
		}

		inline, err := decodeTopicJSON(item, depth+1)
		if err != nil {
			return Topic{}, fmt.Errorf("decode inline subtopic %d: %w", index, err)
		}
		if inline.ID == "" {
			return Topic{}, fmt.Errorf("decode inline subtopic %d: ID is required", index)
		}
		if topic.inlineSubtopics == nil {
			topic.inlineSubtopics = make(map[int]Topic)
		}
		topic.Subtopics = append(topic.Subtopics, inline.ID)
		topic.inlineSubtopics[index] = inline
	}

	return topic, nil
}

func decodeDependencyJSON(data json.RawMessage) (Dependency, error) {
	var shorthand string
	if err := json.Unmarshal(data, &shorthand); err == nil {
		at := strings.LastIndexByte(shorthand, '@')
		if at < 0 {
			return Dependency{}, fmt.Errorf("invalid shorthand %q", shorthand)
		}
		ownerAndName := shorthand[:at]
		owner, name, found := strings.Cut(ownerAndName, "/")
		if !found || strings.Contains(name, "/") {
			return Dependency{}, fmt.Errorf("invalid shorthand %q", shorthand)
		}
		return Dependency{
			Owner:   owner,
			Name:    name,
			Version: shorthand[at+1:],
		}, nil
	}

	var raw rawDependency
	if err := json.Unmarshal(data, &raw); err != nil {
		return Dependency{}, err
	}

	dependency := Dependency{
		Owner:    raw.Owner,
		Name:     raw.Name,
		Version:  raw.Version,
		longForm: true,
	}
	if raw.Locations != nil {
		dependency.Locations = append([]string(nil), (*raw.Locations)...)
		dependency.locationsPresent = true
	}

	return dependency, nil
}

type wireList struct {
	Schema       string               `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Owner        string               `json:"owner" yaml:"owner"`
	Name         string               `json:"name" yaml:"name"`
	Description  string               `json:"description" yaml:"description"`
	Version      string               `json:"version" yaml:"version"`
	IssuedAt     time.Time            `json:"issued-at" yaml:"issued-at"`
	Signature    *string              `json:"signature" yaml:"signature"`
	SignedBy     *string              `json:"signed-by" yaml:"signed-by"`
	Topics       map[string]wireTopic `json:"topics" yaml:"topics"`
	Dependencies map[string]any       `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

type wireTopic struct {
	Schema         string   `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	ID             string   `json:"id,omitempty" yaml:"id,omitempty"`
	DisplayName    string   `json:"display-name,omitempty" yaml:"display-name,omitempty"`
	Description    string   `json:"description" yaml:"description"`
	DocsURL        string   `json:"docs-url,omitempty" yaml:"docs-url,omitempty"`
	ValidityPeriod int      `json:"validity-period,omitempty" yaml:"validity-period,omitempty"`
	Subtopics      []any    `json:"subtopics,omitempty" yaml:"subtopics,omitempty"`
	Pretopics      []string `json:"pretopics,omitempty" yaml:"pretopics,omitempty"`
}

type wireDependency struct {
	Owner     string    `json:"topic-list-owner" yaml:"topic-list-owner"`
	Name      string    `json:"topic-list-name" yaml:"topic-list-name"`
	Version   string    `json:"topic-list-version" yaml:"topic-list-version"`
	Locations *[]string `json:"locations,omitempty" yaml:"locations,omitempty"`
}

type protectedList struct {
	Owner        string                         `json:"owner"`
	Name         string                         `json:"name"`
	Description  string                         `json:"description"`
	Version      string                         `json:"version"`
	IssuedAt     string                         `json:"issued-at"`
	Topics       map[string]protectedTopic      `json:"topics"`
	Dependencies map[string]protectedDependency `json:"dependencies,omitempty"`
}

type protectedTopic struct {
	ID             string   `json:"id,omitempty"`
	DisplayName    string   `json:"display-name,omitempty"`
	Description    string   `json:"description"`
	DocsURL        string   `json:"docs-url,omitempty"`
	ValidityPeriod int      `json:"validity-period,omitempty"`
	Subtopics      []any    `json:"subtopics,omitempty"`
	Pretopics      []string `json:"pretopics,omitempty"`
}

type protectedDependency struct {
	Owner   string `json:"topic-list-owner"`
	Name    string `json:"topic-list-name"`
	Version string `json:"topic-list-version"`
}

type rawList struct {
	Schema       string                     `json:"$schema"`
	Owner        string                     `json:"owner"`
	Name         string                     `json:"name"`
	Description  string                     `json:"description"`
	Version      string                     `json:"version"`
	IssuedAt     time.Time                  `json:"issued-at"`
	Signature    *string                    `json:"signature"`
	SignedBy     *string                    `json:"signed-by"`
	Topics       map[string]json.RawMessage `json:"topics"`
	Dependencies map[string]json.RawMessage `json:"dependencies"`
}

type rawTopic struct {
	Schema         string            `json:"$schema"`
	ID             string            `json:"id"`
	DisplayName    string            `json:"display-name"`
	Description    string            `json:"description"`
	DocsURL        string            `json:"docs-url"`
	ValidityPeriod int               `json:"validity-period"`
	Subtopics      []json.RawMessage `json:"subtopics"`
	Pretopics      []string          `json:"pretopics"`
}

type rawDependency struct {
	Owner     string    `json:"topic-list-owner"`
	Name      string    `json:"topic-list-name"`
	Version   string    `json:"topic-list-version"`
	Locations *[]string `json:"locations"`
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
