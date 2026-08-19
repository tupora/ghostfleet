package bootstrap

import (
	"flag"
	"fmt"
	"io"

	"github.com/tupora/ghostfleet/internal/buildinfo"
)

// Run provides the shared, dependency-free process bootstrap used until each
// command gains its Phase 0 service wiring.
func Run(name string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	showVersion := flags.Bool("version", false, "print version information")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintf(stdout, "%s %s (%s)\n", name, buildinfo.Version, buildinfo.Commit)
		return 0
	}
	fmt.Fprintf(stderr, "%s: no command selected; use -version for build information\n", name)
	return 2
}
