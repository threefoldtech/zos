package fileutils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/threefoldtech/zos/tools/openrpc-codegen/schema"
)

func Parse(filePath string) (spec schema.Spec, err error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return spec, fmt.Errorf("failed to read file content filepath=%v: %w", filePath, err)
	}

	if err := json.Unmarshal(content, &spec); err != nil {
		return spec, fmt.Errorf("failed to unmarshal file content to schema spec: %w", err)
	}

	return spec, nil
}
