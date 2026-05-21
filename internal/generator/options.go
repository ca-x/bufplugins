package generator

import (
	"errors"
	"strings"
)

type Options struct {
	RuntimeImport        string
	FileSuffix           string
	PackageSuffix        string
	ConnectPackageSuffix string
}

func DefaultOptions() Options {
	return Options{
		RuntimeImport:        "github.com/ca-x/bufplugins/runtime/echoadapter",
		FileSuffix:           ".echo.pb.go",
		PackageSuffix:        "echo",
		ConnectPackageSuffix: "connect",
	}
}

func (o Options) Validate() error {
	if strings.TrimSpace(o.RuntimeImport) == "" {
		return errors.New("runtime_import must not be empty")
	}
	return nil
}
