package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config holds the URLs, access tokens, and group paths for source and target GitLab instances
type Config struct {
	SourceGitlabURL   string   `yaml:"source_gitlab_url"`
	TargetGitlabURL   string   `yaml:"target_gitlab_url"`
	SourceAccessToken string   `yaml:"source_access_token"`
	TargetAccessToken string   `yaml:"target_access_token"`
	SourceGroup       string   `yaml:"source_group"`
	TargetGroup       string   `yaml:"target_group"`
	SpecificProjects  []string `yaml:"specific_projects"`
}

// Load reads the YAML configuration file at the given path and returns a Config.
// It applies defaults for target URL and token if they are not provided.
func Load(path string) (*Config, error) {
	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	if err = yaml.Unmarshal(configData, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	if cfg.TargetGitlabURL == "" {
		cfg.TargetGitlabURL = cfg.SourceGitlabURL
	}

	if cfg.TargetAccessToken == "" {
		cfg.TargetAccessToken = cfg.SourceAccessToken
	}

	return &cfg, nil
}
