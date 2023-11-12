package env

import (
	"context"
	"os"
	"path/filepath"

	"github.com/dio/sh"
	"github.com/jdxcode/netrc"
	"github.com/mitchellh/go-homedir"
)

var GH_TOKEN = Var("GH_TOKEN").GetOr(fromNetrc())
var GCLOUD_TOKEN = Var("GCLOUD_TOKEN").GetOr(fromGcloudPrintToken())

type Var string

func (v Var) GetOr(another string) string {
	val := v.Get()
	if len(val) == 0 {
		return another
	}
	return val
}

func (v Var) Get() string {
	return os.Getenv(string(v))
}

func fromNetrc() string {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	parsed, err := netrc.Parse(filepath.Join(home, ".netrc"))
	if err != nil {
		return ""
	}

	return parsed.Machine("github.com").Get("password")
}

func fromGcloudPrintToken() string {
	token, err := sh.Output(context.Background(), "gcloud", "auth", "print-access-token")
	if err != nil {
		return ""
	}
	return token
}
