package utils_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dio/leo/utils"
)

func TestReplaceMatchedLine(t *testing.T) {
	utils.ReplaceMatchedLine(
		filepath.Join("testdata", "Makefile.core.mk"),
		filepath.Join("testdata", "Makefile.core.mk.mod"),
		func(s string) string {
			if strings.Contains(s, "GOOS=linux") && !strings.Contains(s, "GOARM=7") {
				return strings.Replace(s, "GOOS=linux", "GOOS=linux OK=1", 1)
			}
			return s
		},
	)
}
