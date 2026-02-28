package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"

	flagFile    string
	flagEnv     string
	flagDryRun  bool
	flagProfile string
)

var rootCmd = &cobra.Command{
	Use:           "hookdeck-deploy",
	Short:         "Deploy Hookdeck resources from manifest files",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagFile, "file", "f", "", "manifest file path (default: hookdeck.jsonc or hookdeck.json)")
	rootCmd.PersistentFlags().StringVarP(&flagEnv, "env", "e", "", "environment overlay (e.g. staging, production)")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "preview changes without applying")
	rootCmd.PersistentFlags().StringVar(&flagProfile, "profile", "", "override credential profile")
}
