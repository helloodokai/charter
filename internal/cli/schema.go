package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/charter"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Print the JSON schema for charter.yaml",
	Long:  `Print the JSON schema that defines the charter.yaml format. Useful for validation tools and IDE integration.`,
	RunE:  runSchema,
}

var schemaOut string

func init() {
	schemaCmd.Flags().StringVar(&schemaOut, "out", "", "write schema to file instead of stdout")
	rootCmd.AddCommand(schemaCmd)
}

func runSchema(cmd *cobra.Command, args []string) error {
	schema := charter.JSONSchema()
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("generating schema: %w", err)
	}

	if schemaOut != "" {
		return os.WriteFile(schemaOut, data, 0o644)
	}

	fmt.Println(string(data))
	return nil
}