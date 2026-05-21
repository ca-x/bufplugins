package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ca-x/bufplugins/internal/generator"
	"github.com/ca-x/bufplugins/internal/version"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("protoc-gen-echo-v5 %s\n", version.ProtocGenEchoV5)
		return
	}

	var flags flag.FlagSet
	opts := generator.DefaultOptions()
	flags.StringVar(&opts.RuntimeImport, "runtime_import", opts.RuntimeImport, "Echo adapter runtime import path")
	flags.StringVar(&opts.FileSuffix, "file_suffix", opts.FileSuffix, "generated file suffix")
	flags.StringVar(&opts.PackageSuffix, "package_suffix", opts.PackageSuffix, "Echo generated package suffix")
	flags.StringVar(&opts.ConnectPackageSuffix, "connect_package_suffix", opts.ConnectPackageSuffix, "connect-go package suffix")

	protogen.Options{ParamFunc: flags.Set}.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL) | uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)
		plugin.SupportedEditionsMinimum = descriptorpb.Edition_EDITION_PROTO2
		plugin.SupportedEditionsMaximum = descriptorpb.Edition_EDITION_2024
		for _, file := range plugin.Files {
			if file.Generate {
				if err := generator.GenerateFile(plugin, file, opts); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
