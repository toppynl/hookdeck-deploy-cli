package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toppynl/hookdeck-deploy-cli/schemas"
)

var projectFlag bool

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Output JSON schema for manifest files",
	Args:  cobra.NoArgs,
	RunE:  runSchema,
}

func init() {
	schemaCmd.Flags().BoolVar(&projectFlag, "project", false, "Output the project configuration schema instead of the deploy schema")
	rootCmd.AddCommand(schemaCmd)
}

func runSchema(cmd *cobra.Command, args []string) error {
	if projectFlag {
		fmt.Print(schemas.ProjectSchema)
	} else {
		fmt.Print(schemas.DeploySchema)
	}
	return nil
}
