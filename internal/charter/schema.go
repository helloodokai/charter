package charter

import (
	"encoding/json"
	"os"
)

// EmitJSONSchema writes the JSON Schema for the Charter type to the given file path.
func EmitJSONSchema(path string) error {
	schema := JSONSchema()
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}