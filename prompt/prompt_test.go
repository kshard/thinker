package prompt_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/fogfish/it/v2"
	"github.com/kshard/thinker/prompt"
)

// -----------------------------------------------------------------------------
// Parse
// -----------------------------------------------------------------------------

func TestParseEmpty(t *testing.T) {
	it.Then(t).Should(
		it.Error(prompt.Parse(strings.NewReader(""))),
	)
}

func TestParsePurePlaintext(t *testing.T) {
	const text = "What is the capital of {{.Country}}?"

	p, err := prompt.Parse(strings.NewReader(text))

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(p.Prompt, text),
		it.Equal(p.RunsOn, ""),
		it.Equal(p.Schema.Format, ""),
	).Should(
		it.Seq(p.Servers).BeEmpty(),
	)
}

func TestParseOnlyOpenDelimiter(t *testing.T) {
	// A file that starts with --- but has no closing --- is treated as a plain prompt
	const text = "---\nsome prompt text without closing delimiter\n"

	p, err := prompt.Parse(strings.NewReader(text))

	it.Then(t).Should(
		it.Nil(err),
		it.String(p.Prompt).Contain("some prompt text"),
	)
}

func TestParseTooManyDelimiters(t *testing.T) {
	const text = "---\nfoo: bar\n---\nprompt text\n---\nextra section\n"

	it.Then(t).Should(
		it.Error(prompt.Parse(strings.NewReader(text))),
	)
}

func TestParseInvalidYAML(t *testing.T) {
	const text = "---\n: invalid: yaml: [\n---\nsome prompt\n"

	it.Then(t).Should(
		it.Error(prompt.Parse(strings.NewReader(text))),
	)
}

func TestParseFrontmatterMinimal(t *testing.T) {
	const text = "---\nruns-on: medium\n---\nAnswer the question.\n"

	p, err := prompt.Parse(strings.NewReader(text))

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(p.RunsOn, "medium"),
		it.Equal(p.Prompt, "Answer the question.\n"),
	).Should(
		it.Seq(p.Servers).BeEmpty(),
	)
}

func TestParseFrontmatterFormat(t *testing.T) {
	const text = "---\nformat: json\n---\nReturn JSON.\n"

	p, err := prompt.Parse(strings.NewReader(text))

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(p.Schema.Format, "json"),
	)
}

func TestParseFrontmatterWithSchema(t *testing.T) {
	const text = "---\nformat: json\nruns-on: small\nschema:\n  input:\n    type: object\n    properties:\n      country:\n        type: string\n  reply:\n    type: object\n    properties:\n      capital:\n        type: string\n---\nWhat is the capital of {{.Country}}?\n"

	p, err := prompt.Parse(strings.NewReader(text))

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(p.RunsOn, "small"),
		it.Equal(p.Schema.Format, "json"),
		it.String(p.Prompt).Contain("capital of {{.Country}}"),
	).ShouldNot(
		it.Nil(p.Schema.Input),
		it.Nil(p.Schema.Reply),
	)
}

func TestParseFrontmatterWithServer(t *testing.T) {
	const text = "---\nservers:\n  - type: mcp\n    name: myserver\n    command: [mycommand, --port, \"8080\"]\n    url: http://localhost:8080\n---\nUse the server.\n"

	p, err := prompt.Parse(strings.NewReader(text))

	it.Then(t).Should(
		it.Nil(err),
	).Should(
		it.Seq(p.Servers).Equal(prompt.Server{
			Type:    "mcp",
			Name:    "myserver",
			Command: []string{"mycommand", "--port", "8080"},
			Url:     "http://localhost:8080",
		}),
	)
}

func TestParseFrontmatterMultipleServers(t *testing.T) {
	const text = "---\nservers:\n  - type: mcp\n    name: alpha\n    command: [cmd-a]\n  - type: stdio\n    name: beta\n    command: [cmd-b, --verbose]\n---\nUse multiple servers.\n"

	p, err := prompt.Parse(strings.NewReader(text))

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(len(p.Servers), 2),
		it.Equal(p.Servers[0].Name, "alpha"),
		it.Equal(p.Servers[1].Name, "beta"),
	)
}

// -----------------------------------------------------------------------------
// ParseFile
// -----------------------------------------------------------------------------

func TestParseFileSuccess(t *testing.T) {
	const content = "---\nruns-on: large\n---\nDescribe the image.\n"

	fsys := fstest.MapFS{
		"prompts/describe.md": &fstest.MapFile{Data: []byte(content)},
	}

	p, err := prompt.ParseFile(fsys, "prompts/describe.md")

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(p.RunsOn, "large"),
		it.String(p.Prompt).Contain("Describe the image"),
	)
}

func TestParseFileNotFound(t *testing.T) {
	fsys := fstest.MapFS{}

	it.Then(t).Should(
		it.Error(prompt.ParseFile(fsys, "missing.md")),
	)
}

func TestParseFilePlaintext(t *testing.T) {
	const content = "Translate the following text.\n"

	fsys := fstest.MapFS{
		"translate.md": &fstest.MapFile{Data: []byte(content)},
	}

	p, err := prompt.ParseFile(fsys, "translate.md")

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(p.Prompt, content),
	)
}
