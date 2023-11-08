package istio_test

import (
	"os"
	"testing"

	"github.com/dio/leo/istio"
)

func TestBaseImage(t *testing.T) {
	data, _ := os.ReadFile("testdata/Makefile.core.mk")
	extracted, err := istio.BaseImageFromMakefileCore(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if extracted.Registry != "gcr.io/istio-release" {
		t.Fatal("invalid registry", extracted.Registry)
	}
	if extracted.Version != "master-2023-10-12T19-01-47" {
		t.Fatal("invalid version")
	}
}
