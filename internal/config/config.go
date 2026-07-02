package config

import (
	"encoding/json"
	"os"
	"sort"
)

type Config struct {
	Handlers []HandlerDef `json:"handlers"`
}

type HandlerDef struct {
	ID          string                `json:"id"`
	File        string                `json:"file"`
	Function    string                `json:"function"`
	Method      string                `json:"method"`
	Path        string                `json:"path"`
	Summary     string                `json:"summary"`
	Description string                `json:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Consumes    string                `json:"consumes,omitempty"` // Accept
	Produces    string                `json:"produces,omitempty"` // Produce
	Parameters  []ParameterDef        `json:"parameters,omitempty"`
	Responses   map[string]Response   `json:"responses,omitempty"`
	Security    []map[string][]string `json:"security,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty"`
}

type ParameterDef struct {
	Name        string     `json:"name"`
	In          string     `json:"in"`
	Required    bool       `json:"required"`
	Schema      *SchemaDef `json:"schema,omitempty"`
	Type        string     `json:"type,omitempty"` // упрощённый вариант
	Description string     `json:"description,omitempty"`
}

type SchemaDef struct {
	Type       string                 `json:"type,omitempty"`
	Ref        string                 `json:"$ref,omitempty"`
	Items      *SchemaDef             `json:"items,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type Response struct {
	Description string     `json:"description,omitempty"`
	Schema      *SchemaDef `json:"schema,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Handlers: []HandlerDef{}}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	sort.Slice(cfg.Handlers, func(i, j int) bool {
		return cfg.Handlers[i].ID < cfg.Handlers[j].ID
	})
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
