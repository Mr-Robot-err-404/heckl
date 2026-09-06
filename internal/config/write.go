package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed config.toml.tmpl
var configTemplate string

func (c *Config) Save(path string) error {
	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return fmt.Errorf("config: parse template: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: create %s: %w", filepath.Dir(path), err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, c); err != nil {
		return fmt.Errorf("config: render %s: %w", path, err)
	}
	c.path = path
	return nil
}
