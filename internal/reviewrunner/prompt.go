package reviewrunner

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"orchv3/internal/agentmeta"
)

//go:embed prompts/*.tmpl
var defaultPromptFS embed.FS

type PromptInput struct {
	Stage         agentmeta.Stage
	ProducerBy    string
	ProducerModel string
	ReviewerBy    string
	ReviewerModel string
	Categories    []string
	Targets       []Target
}

// RenderPrompt resolves the template for the input's stage and renders it
// with the PromptInput. When overrideDir is non-empty and contains a file
// named after the stage's PromptName, that file is used instead of the embedded default.
func RenderPrompt(in PromptInput, overrideDir string) (string, error) {
	profile, err := ProfileFor(in.Stage)
	if err != nil {
		return "", err
	}

	tmplBytes, err := loadTemplate(profile.PromptName, overrideDir)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(profile.PromptName).Parse(string(tmplBytes))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", profile.PromptName, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, in); err != nil {
		return "", fmt.Errorf("execute template %s: %w", profile.PromptName, err)
	}
	return buf.String(), nil
}

func loadTemplate(name, overrideDir string) ([]byte, error) {
	if overrideDir != "" {
		path := filepath.Join(overrideDir, name)
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read override template %s: %w", path, err)
		}
	}
	return defaultPromptFS.ReadFile("prompts/" + name)
}
