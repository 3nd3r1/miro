package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithRunner(context.Background(), args, ExecRunner{}, stdout, stderr)
}

func runWithRunner(ctx context.Context, args []string, runner Runner, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 1
	}

	switch args[0] {
	case "build":
		return runBuild(ctx, args[1:], runner, stdout, stderr)
	case "export":
		return runExport(ctx, args[1:], runner, stdout, stderr)
	case "import-status":
		return runImportStatus(args[1:], runner, stdout, stderr)
	case "status":
		return runStatus(args[1:], runner, stdout, stderr)
	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "miro %s\n", version)
		return 0
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "miro: unknown command %q\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func resolveStatePath(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if env := os.Getenv("MIRO_STATE"); env != "" {
		return env
	}
	return defaultStatePath
}

func runBuild(ctx context.Context, args []string, runner Runner, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	flakeRef := fs.String("flake", ".", "flake reference")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: miro build [--flake <path>] <site>")
		return 1
	}

	hub := &Hub{Runner: runner}
	outPath, err := hub.Build(ctx, *flakeRef, fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "miro build: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, outPath)
	return 0
}

func runExport(ctx context.Context, args []string, runner Runner, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	statePath := fs.String("state", "", "path to hub state file")
	flakeRef := fs.String("flake", ".", "flake reference")
	systemPath := fs.String("system", "", "pre-built NixOS system store path (skips nix build)")
	baseSystem := fs.String("base-system", "", "optional base system store path")
	modelPath := fs.String("model", "", "path to GGUF model file to include in models/<sha256>")
	modelName := fs.String("model-name", "", "optional display name for model artifact")
	bundleID := fs.String("bundle-id", "", "optional bundle ID override")
	compression := fs.String("compression", "none", "binary cache NAR compression (none, xz, zstd)")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(stderr, "usage: miro export [flags] <site> <outdir>")
		return 1
	}

	hub := &Hub{
		StatePath: resolveStatePath(*statePath),
		Runner:    runner,
	}
	m, err := hub.Export(ctx, ExportOptions{
		Site:        fs.Arg(0),
		OutDir:      fs.Arg(1),
		FlakeRef:    *flakeRef,
		SystemPath:  *systemPath,
		BaseSystem:  *baseSystem,
		ModelPath:   *modelPath,
		ModelName:   *modelName,
		BundleID:    *bundleID,
		Compression: *compression,
	})
	if err != nil {
		fmt.Fprintf(stderr, "miro export: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "exported %s (site=%s seq=%d system=%s)\n", m.BundleID, m.SiteID, m.Sequence, m.TargetSystem)
	return 0
}

func runImportStatus(args []string, runner Runner, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("import-status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	statePath := fs.String("state", "", "path to hub state file")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: miro import-status [--state <path>] <dir>")
		return 1
	}

	hub := &Hub{
		StatePath: resolveStatePath(*statePath),
		Runner:    runner,
	}
	sb, err := hub.ImportStatus(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "miro import-status: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "imported status for %s (seq=%d gen=%d healthy=%t)\n",
		sb.SiteID, sb.LastAppliedSequence, sb.CurrentGeneration, sb.Health.Healthy)
	return 0
}

func runStatus(args []string, runner Runner, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	statePath := fs.String("state", "", "path to hub state file")
	asJSON := fs.Bool("json", false, "output hub state as JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: miro status [--state <path>] [--json]")
		return 1
	}

	hub := &Hub{
		StatePath: resolveStatePath(*statePath),
		Runner:    runner,
	}
	if err := hub.WriteStatus(stdout, *asJSON); err != nil {
		fmt.Fprintf(stderr, "miro status: %v\n", err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: miro <command> [flags] [args]")
	fmt.Fprintln(w, "commands: build, export, import-status, status, version, help")
}
