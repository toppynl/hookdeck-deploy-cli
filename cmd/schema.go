package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toppynl/hookdeck-deploy-cli/schemas"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Output JSON schema for manifest files",
	Args:  cobra.NoArgs,
	RunE:  runSchema,
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}

func runSchema(cmd *cobra.Command, args []string) error {
	fmt.Print(schemas.DeploySchema)
	return nil
}
