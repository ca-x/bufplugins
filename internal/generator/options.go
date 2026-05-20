package generator

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
