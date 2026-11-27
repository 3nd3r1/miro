package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithDeps(context.Background(), args, ExecRunner{}, HTTPHealthChecker{}, stdout, stderr)
}

func runWithDeps(
	ctx context.Context,
	args []string,
	runner Runner,
	health HealthChecker,
	stdout, stderr io.Writer,
) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 1
	}

	switch args[0] {
	case "reconcile":
		return runReconcile(ctx, args[1:], runner, health, stdout, stderr)
	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "miro-agent %s\n", version)
		return 0
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "miro-agent: unknown command %q\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func runReconcile(
	ctx context.Context,
	args []string,
	runner Runner,
	health HealthChecker,
	stdout, stderr io.Writer,
) int {
	fs := flag.NewFlagSet("reconcile", flag.ContinueOnError)
	fs.SetOutput(stderr)

	siteID := fs.String("site", os.Getenv("MIRO_SITE_ID"), "site identifier")
	inboundDir := fs.String("inbound", "/var/lib/miro/transfer/inbound", "inbound bundles directory")
	statusOutDir := fs.String("status-out", "/var/lib/miro/transfer/status-out", "outbound status directory")
	statePath := fs.String("state", "/var/lib/miro/agent-state.json", "persisted agent state file")
	modelsDir := fs.String("models-dir", "/var/lib/miro/models", "content-addressed models directory")
	systemProfile := fs.String("system-profile", "/nix/var/nix/profiles/system", "NixOS system profile path")
	healthURL := fs.String("health-url", "http://127.0.0.1:8080/health", "local inference health URL")
	healthTimeout := fs.Duration("health-timeout", 60*time.Second, "timeout waiting for inference health")
	healthInterval := fs.Duration("health-interval", time.Second, "poll interval for inference health")
	inferenceUnit := fs.String("inference-unit", "llama-cpp.service", "systemd unit to ensure started after switch")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: miro-agent reconcile [flags]")
		return 1
	}

	agent := &Agent{
		Cfg: Config{
			SiteID:         *siteID,
			InboundDir:     *inboundDir,
			StatusOutDir:   *statusOutDir,
			StatePath:      *statePath,
			ModelsDir:      *modelsDir,
			SystemProfile:  *systemProfile,
			HealthURL:      *healthURL,
			HealthTimeout:  *healthTimeout,
			HealthInterval: *healthInterval,
			InferenceUnit:  *inferenceUnit,
		},
		Runner: runner,
		Health: health,
		LogOut: stdout,
	}
	if err := agent.Reconcile(ctx); err != nil {
		fmt.Fprintf(stderr, "miro-agent reconcile: %v\n", err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: miro-agent <command> [flags]")
	fmt.Fprintln(w, "commands: reconcile, version, help")
}
