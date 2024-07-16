package fileutils

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
)

func Write(buf bytes.Buffer, filePath string) error {
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format buffer content: %w", err)
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to open file for writing filepath=%v: %w", filePath, err)
	}
	defer file.Close()

	if _, err := file.Write(formatted); err != nil {
		return fmt.Errorf("failed to write on file: %w", err)
	}

	return nil
}
