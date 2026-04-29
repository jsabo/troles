package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	apiclient "github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/profile"

	"github.com/jsabo/troles/internal/roles"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		proxyAddr     string
		cluster       string
		tshProfileDir string
		format        string
		showVersion   bool
	)

	flag.StringVar(&proxyAddr, "proxy", "", "Teleport proxy address (default: from active tsh profile)")
	flag.StringVar(&cluster, "cluster", "", "tsh profile name (proxy host) to use (default: active profile)")
	flag.StringVar(&tshProfileDir, "tsh-profile-dir", "", "tsh profile directory (default: ~/.tsh)")
	flag.StringVar(&format, "format", "table", "Output format: table|json")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("troles %s (%s, %s)\n", version, shorten(commit), date)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	username := flag.Arg(0)
	if username == "" {
		p, err := profile.FromDir(tshProfileDir, cluster)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: could not detect current user from tsh profile: %v\n", err)
			fmt.Fprintf(os.Stderr, "usage: troles [flags] [username]\n")
			os.Exit(1)
		}
		username = p.Username
	}

	clientCfg := apiclient.Config{
		Credentials: []apiclient.Credentials{apiclient.LoadProfile(tshProfileDir, cluster)},
	}
	if proxyAddr != "" {
		clientCfg.Addrs = []string{proxyAddr}
	}

	tc, err := apiclient.New(ctx, clientCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: connecting to Teleport: %v\n", err)
		os.Exit(1)
	}
	defer tc.Close()

	result, err := roles.Get(ctx, tc, username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch format {
	case "json":
		if err := result.PrintJSON(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		result.PrintTable(os.Stdout)
	}
}

func shorten(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
