package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/toppynl/hookdeck-deploy-cli/schemas"
)

var schemaCmd = &cobra.Command{
	Use:   "schema [deploy|transformation]",
	Short: "Output JSON schema for manifest files",
	Args:  cobra.ExactArgs(1),
	RunE:  runSchema,
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}

func runSchema(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "deploy":
		fmt.Print(schemas.DeploySchema)
	case "transformation":
		fmt.Print(schemas.TransformationSchema)
	default:
		return fmt.Errorf("unknown schema: %s (use 'deploy' or 'transformation')", args[0])
	}
	return nil
}
