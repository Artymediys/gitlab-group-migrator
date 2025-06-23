package main

import (
	"flag"
	"fmt"
	"os"

	"gitlab-group-migrator/internal/config"
	"gitlab-group-migrator/internal/gitlab"
)

// main is the entry point of the application.
// It executes the run function and handles error reporting.
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migration completed successfully.")
}

// run performs the primary workflow: parsing flags, loading configuration,
// fetching source and target groups, and initiating the namespace migration.
func run() error {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("loading config from %s: %w", *cfgPath, err)
	}

	sourceGroup, err := gitlab.FetchGroup(cfg.SourceGitlabURL, cfg.SourceAccessToken, cfg.SourceGroup)
	if err != nil {
		return fmt.Errorf("fetching source group %s: %w", cfg.SourceGroup, err)
	}

	targetGroup, err := gitlab.FetchGroup(cfg.TargetGitlabURL, cfg.TargetAccessToken, cfg.TargetGroup)
	if err != nil {
		return fmt.Errorf("fetching target group %s: %w", cfg.TargetGroup, err)
	}

	if err = gitlab.MigrateNamespace(cfg, sourceGroup.ID, targetGroup.ID); err != nil {
		return fmt.Errorf("migrating namespace %s -> %s: %w", cfg.SourceGroup, cfg.TargetGroup, err)
	}

	return nil
}
