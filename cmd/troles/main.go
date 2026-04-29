package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		verbose       bool
		showVersion   bool
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Show a Teleport user's effective roles, including access-list grants.\n")
		fmt.Fprintf(os.Stderr, "If username is omitted, uses the currently logged-in tsh user.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: troles [flags] [username]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\ntsh alias:\n")
		fmt.Fprintf(os.Stderr, "  Add to ~/.tsh/config/config.yaml:\n\n")
		fmt.Fprintf(os.Stderr, "    aliases:\n")
		fmt.Fprintf(os.Stderr, "      roles: troles\n\n")
		fmt.Fprintf(os.Stderr, "  Then: tsh roles [username]\n")
	}

	flag.StringVar(&proxyAddr, "proxy", "", "Teleport proxy address (default: from active tsh profile)")
	flag.StringVar(&cluster, "cluster", "", "tsh profile name (proxy host) to use (default: active profile)")
	flag.StringVar(&tshProfileDir, "tsh-profile-dir", "", "tsh profile directory (default: ~/.tsh)")
	flag.StringVar(&format, "format", "table", "Output format: table|json")
	flag.BoolVar(&verbose, "verbose", false, "Print full connection error detail")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("troles %s (%s, %s)\n", version, shorten(commit), date)
		return
	}

	if tshProfileDir == "" {
		tshProfileDir = os.Getenv("TELEPORT_HOME")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	username := flag.Arg(0)
	if username == "" {
		p, err := profile.FromDir(tshProfileDir, cluster)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: not logged in — run 'tsh login' first\n")
			os.Exit(1)
		}
		username = p.Username
	}

	creds := apiclient.LoadProfile(tshProfileDir, cluster)

	if exp, ok := creds.Expiry(); ok && !exp.IsZero() && time.Now().After(exp) {
		fmt.Fprintf(os.Stderr, "error: tsh credentials expired — run 'tsh login' and try again\n")
		os.Exit(1)
	}

	clientCfg := apiclient.Config{
		Credentials: []apiclient.Credentials{creds},
	}
	if proxyAddr != "" {
		clientCfg.Addrs = []string{proxyAddr}
	}

	tc, err := apiclient.New(ctx, clientCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not connect to Teleport — run 'tsh login' and try again\n")
		if verbose {
			fmt.Fprintf(os.Stderr, "\n%v\n", err)
		}
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
		result.PrintTable(os.Stdout, isTTY())
	}
}

func isTTY() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func shorten(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
