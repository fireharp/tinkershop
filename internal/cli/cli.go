package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/fireharp/tinkershop/internal/config"
	"github.com/fireharp/tinkershop/internal/daemon"
	"github.com/fireharp/tinkershop/internal/policy"
	"github.com/fireharp/tinkershop/internal/scan"
	"github.com/fireharp/tinkershop/internal/server"
)

func Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usage()
	}

	switch args[0] {
	case "scan":
		return runScan(ctx, args[1:])
	case "daemon":
		return runDaemon(ctx, args[1:])
	case "serve":
		return runServe(ctx, args[1:])
	case "policy":
		return runPolicy(args[1:])
	default:
		return usage()
	}
}

func runScan(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to JSON config")
	dbPath := fs.String("db", "", "SQLite path override")
	blobDir := fs.String("blob-dir", "", "blob directory override")
	var roots stringList
	fs.Var(&roots, "root", "scan root; repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := load(*configPath, *dbPath, *blobDir, roots)
	if err != nil {
		return err
	}
	summary, err := scan.Run(ctx, cfg)
	if err != nil {
		return err
	}
	fmt.Printf("run=%d projects=%d observations=%d\n", summary.RunID, summary.ProjectCount, summary.ObservationCount)
	return nil
}

func runDaemon(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to JSON config")
	dbPath := fs.String("db", "", "SQLite path override")
	blobDir := fs.String("blob-dir", "", "blob directory override")
	interval := fs.Duration("interval", 12*time.Hour, "scan interval")
	var roots stringList
	fs.Var(&roots, "root", "scan root; repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := load(*configPath, *dbPath, *blobDir, roots)
	if err != nil {
		return err
	}
	return daemon.Run(ctx, cfg, *interval)
}

func runServe(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to JSON config")
	dbPath := fs.String("db", "", "SQLite path override")
	blobDir := fs.String("blob-dir", "", "blob directory override")
	addr := fs.String("addr", "127.0.0.1:8739", "listen address")
	var roots stringList
	fs.Var(&roots, "root", "scan root; repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := load(*configPath, *dbPath, *blobDir, roots)
	if err != nil {
		return err
	}
	return server.ListenAndServe(ctx, cfg, *addr)
}

func runPolicy(args []string) error {
	fs := flag.NewFlagSet("policy", flag.ContinueOnError)
	path := fs.String("path", "", "path to evaluate")
	configPath := fs.String("config", "", "path to JSON config")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("-path is required")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	decision := policy.Evaluate(*path, cfg.Policies)
	fmt.Printf("%s", decision.State)
	if decision.DisplayName != "" {
		fmt.Printf(" %q", decision.DisplayName)
	}
	fmt.Println()
	return nil
}

func load(configPath, dbPath, blobDir string, roots []string) (config.Config, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return config.Config{}, err
	}
	if dbPath != "" {
		cfg.DBPath = dbPath
	}
	if blobDir != "" {
		cfg.BlobDir = blobDir
	}
	if len(roots) > 0 {
		cfg.Roots = roots
	}
	return cfg, cfg.Validate()
}

func usage() error {
	return fmt.Errorf("usage: tinkershop <scan|daemon|serve|policy>")
}

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}
