package prompt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/google/jsonschema-go/jsonschema"
)

// Prompt is a markdown document with YAMLs based metadata, it containts:
// * prompt itself as a temaplate with variables
// * expected input/output format (json schema)
// * list of servers
type Prompt struct {
	// Prompt template, using Golang template syntax, e.g. "What is the capital of {{.Country}}?"
	Prompt string

	// Optional field to specify which LLM to use, e.g. "base", "micro", "small", "medium", "large", "xlarge"
	RunsOn string

	// Optional field to specify how many times to retry the prompt in case of failure.
	// Default is 3
	Retry int

	// Optional debugging mode
	Debug bool

	// Input/ Output schema
	Schema Schema

	// List of servers required for the task
	Servers []Server
}

// Input and output schema, using JSON Schema format, e.g. {"type": "object", "properties": {"Country": {"type": "string"}}}
type Schema struct {
	Format string
	Input  *jsonschema.Schema
	Reply  *jsonschema.Schema
}

// Server specification
type Server struct {
	Type    string
	Name    string
	Command []string
	Url     string
}

type yamlPrompt struct {
	Format  string       `yaml:"format,omitempty"`
	RunsOn  string       `yaml:"runs-on,omitempty"`
	Retry   int          `yaml:"retry,omitempty"`
	Debug   bool         `yaml:"debug,omitempty"`
	Schema  *yamlSchema  `yaml:"schema,omitempty"`
	Servers []yamlServer `yaml:"servers,omitempty"`
}

type yamlSchema struct {
	Input map[string]any `yaml:"input,omitempty"`
	Reply map[string]any `yaml:"reply,omitempty"`
}

type yamlServer struct {
	Type    string   `yaml:"type,omitempty"`
	Name    string   `yaml:"name,omitempty"`
	Command []string `yaml:"command,omitempty"`
	Url     string   `yaml:"url,omitempty"`
}

// Build a prompt from a markdown file, the file might contain YAML metadata and must markdown content
func ParseFile(fs fs.FS, path string) (*Prompt, error) {
	fd, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open agent file: %w", err)
	}
	defer fd.Close()

	return Parse(fd)
}

// Build a prompt from a markdown content, the content might contain YAML metadata and must markdown content
func Parse(r io.Reader) (*Prompt, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("prompt file is empty")
	}

	var div = []byte("---\n")

	if !bytes.HasPrefix(data, div) {
		return &Prompt{Prompt: string(data)}, nil
	}

	parts := bytes.SplitN(data[len(div):], []byte(div), 2)

	// Only starting delimiter, no ending delimiter - treat as pure prompt
	if len(parts) == 1 {
		return &Prompt{Prompt: string(parts[0])}, nil
	}

	// Has frontmatter and prompt
	var raw yamlPrompt
	if err := yaml.Unmarshal(parts[0], &raw); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if len(raw.Format) == 0 {
		raw.Format = "text"
	}

	if len(raw.RunsOn) == 0 {
		raw.RunsOn = "base"
	}

	return toPrompt(&raw, string(parts[1])), nil
}

func toPrompt(raw *yamlPrompt, prompt string) *Prompt {
	servers := make([]Server, 0, len(raw.Servers))
	for _, srv := range raw.Servers {
		argv := make([]string, len(srv.Command))
		for i, arg := range srv.Command {
			argv[i] = os.ExpandEnv(arg)
		}

		servers = append(servers, Server{
			Type:    srv.Type,
			Name:    srv.Name,
			Command: argv,
			Url:     srv.Url,
		})
	}

	// Handle optional schema
	var inputSchema, replySchema *jsonschema.Schema
	if raw.Schema != nil {
		inputSchema = toSchema(raw.Schema.Input)
		replySchema = toSchema(raw.Schema.Reply)
	}

	if raw.Retry == 0 {
		raw.Retry = 3
	}

	return &Prompt{
		Prompt: prompt,
		RunsOn: raw.RunsOn,
		Retry:  raw.Retry,
		Debug:  raw.Debug,
		Schema: Schema{
			Format: raw.Format,
			Input:  inputSchema,
			Reply:  replySchema,
		},
		Servers: servers,
	}
}

func toSchema(schema map[string]any) *jsonschema.Schema {
	if schema == nil {
		return nil
	}

	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil
	}

	var js jsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &js); err != nil {
		return nil
	}

	return &js
}
