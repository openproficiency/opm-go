// Package schema embeds and validates against the OPM v0.1.1 schemas.
package schema

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/openproficiency/opm-go/internal/semantic"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

const remoteSchemaBase = "https://raw.githubusercontent.com/openproficiency/model/refs/heads/main/schemas/"

// Version is the bundled Open Proficiency Model release.
const Version = "v0.1.1"

// Name identifies an embedded OPM schema.
type Name string

const (
	ScoreInterpretationList     Name = "score-interpretation-list.schema.json"
	ScoreInterpretation         Name = "score-interpretation.schema.json"
	TopicList                   Name = "topic-list.schema.json"
	Topic                       Name = "topic.schema.json"
	TranscriptEntryVerification Name = "transcript-entry-verification.schema.json"
	TranscriptEntry             Name = "transcript-entry.schema.json"
	Transcript                  Name = "transcript.schema.json"
)

var (
	//go:embed assets/*.json
	schemaAssets embed.FS

	compileOnce sync.Once
	compiled    map[Name]*jsonschema.Schema
	compileErr  error
)

var schemaNames = [...]Name{
	ScoreInterpretationList,
	ScoreInterpretation,
	TopicList,
	Topic,
	TranscriptEntryVerification,
	TranscriptEntry,
	Transcript,
}

// Compile returns an offline-compiled draft 2020-12 OPM schema.
func Compile(name Name) (*jsonschema.Schema, error) {
	compileOnce.Do(compileSchemas)

	if compileErr != nil {
		return nil, compileErr
	}

	compiledSchema, exists := compiled[name]
	if !exists {
		return nil, fmt.Errorf("unknown OPM schema %q", name)
	}

	return compiledSchema, nil
}

// Validate checks a decoded JSON-compatible document against an OPM schema.
func Validate(name Name, document any) error {
	compiledSchema, err := Compile(name)
	if err != nil {
		return err
	}
	if err := compiledSchema.Validate(document); err != nil {
		return fmt.Errorf("validate %s: %w", name, err)
	}

	return nil
}

// ValidateJSON checks JSON bytes against an OPM schema.
func ValidateJSON(name Name, data []byte) error {
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode JSON for %s: %w", name, err)
	}

	return Validate(name, document)
}

// ValidateYAML checks YAML bytes against an OPM schema.
func ValidateYAML(name Name, data []byte) error {
	var yamlDocument any
	if err := yaml.Unmarshal(data, &yamlDocument); err != nil {
		return fmt.Errorf("decode YAML for %s: %w", name, err)
	}

	jsonData, err := json.Marshal(yamlDocument)
	if err != nil {
		return fmt.Errorf("normalize YAML for %s: %w", name, err)
	}

	return ValidateJSON(name, jsonData)
}

func compileSchemas() {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(offlineLoader{})
	registerFormats(compiler)

	for _, name := range schemaNames {
		document, err := readSchema(name)
		if err != nil {
			compileErr = err
			return
		}
		if err := compiler.AddResource(remoteSchemaBase+string(name), document); err != nil {
			compileErr = fmt.Errorf("register embedded schema %s: %w", name, err)
			return
		}
	}

	compiled = make(map[Name]*jsonschema.Schema, len(schemaNames))
	for _, name := range schemaNames {
		compiledSchema, err := compiler.Compile(remoteSchemaBase + string(name))
		if err != nil {
			compileErr = fmt.Errorf("compile embedded schema %s: %w", name, err)
			return
		}
		compiled[name] = compiledSchema
	}
}

func readSchema(name Name) (any, error) {
	data, err := schemaAssets.ReadFile("assets/" + string(name))
	if err != nil {
		return nil, fmt.Errorf("read embedded schema %s: %w", name, err)
	}

	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode embedded schema %s: %w", name, err)
	}

	return document, nil
}

func registerFormats(compiler *jsonschema.Compiler) {
	compiler.RegisterFormat(stringFormat("email", semantic.Email))
	compiler.RegisterFormat(stringFormat("hostname", semantic.Hostname))
	compiler.RegisterFormat(stringFormat("kebab-case", semantic.KebabCase))
	compiler.RegisterFormat(stringFormat("semver", semantic.Semver))
	compiler.RegisterFormat(stringFormat("uri", semantic.URI))
}

func stringFormat(name string, validate func(string) error) *jsonschema.Format {
	return &jsonschema.Format{
		Name: name,
		Validate: func(value any) error {
			text, ok := value.(string)
			if !ok {
				return nil
			}

			return validate(text)
		},
	}
}

type offlineLoader struct{}

func (offlineLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external schema loading is disabled: %s", url)
}
