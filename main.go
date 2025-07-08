package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gitlab-group-migrator/internal/config"
	"gitlab-group-migrator/internal/gitlab"
)

// main is the entry point of the application.
// It executes the run function and handles error reporting.
func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	if err := run(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migration completed successfully.")
}

// run performs the primary workflow: parsing flags, loading configuration,
// fetching source and target groups, and initiating the namespace migration.
func run(logger *log.Logger) error {
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

	logger.Printf("Fetched source group %s (ID %d)", sourceGroup.FullPath, sourceGroup.ID)

	targetGroup, err := gitlab.FetchGroup(cfg.TargetGitlabURL, cfg.TargetAccessToken, cfg.TargetGroup)
	if err != nil {
		return fmt.Errorf("fetching target group %s: %w", cfg.TargetGroup, err)
	}

	logger.Printf("Fetched target group %s (ID %d)", targetGroup.FullPath, targetGroup.ID)

	gitlab.MigrateNamespace(cfg, logger, sourceGroup.ID, targetGroup.ID)

	return nil
}
