package test

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

func WriteFile(t *testing.T, dest string, contents string) error {
	t.Helper()

	if runtime.GOOS != "windows" {
		tmpFile := fmt.Sprintf("%s.tmp", dest)
		if err := os.WriteFile(tmpFile, []byte(contents), 0644); err != nil {
			return err
		}

		return os.Rename(tmpFile, dest)
	} else {
		return os.WriteFile(dest, []byte(contents), 0644)
	}
}
