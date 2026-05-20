# bufplugins

Buf/protoc plugins for generating framework adapters around `connect-go` services.

The first plugin is `protoc-gen-echo-v5`. It reads `google.api.http` annotations, generates Echo v5 REST route registration, and keeps the generated Connect handler mounted for Connect/gRPC/gRPC-Web clients.

## Install

```sh
go install github.com/ca-x/bufplugins/cmd/protoc-gen-echo-v5@latest
```

For local development from this repository:

```sh
go install ./cmd/protoc-gen-echo-v5
```

`protoc-gen-echo-v5` must be on `PATH` when `buf generate` runs.

## Buf Configuration

Use the normal Go and Connect plugins plus the Echo plugin:

```yaml
version: v2
managed:
  enabled: true
  disable:
    - file_option: go_package_prefix
      module: buf.build/googleapis/googleapis
    - file_option: go_package_prefix
      module: buf.build/bufbuild/protovalidate
  override:
    - file_option: go_package_prefix
      value: github.com/acme/project/api
    - file_option: go_package
      module: buf.build/bufbuild/protovalidate
      value: buf.build/go/protovalidate
plugins:
  - remote: buf.build/protocolbuffers/go
    out: api
    opt:
      - paths=source_relative
  - remote: buf.build/connectrpc/go
    out: api
    opt:
      - paths=source_relative
  - local: protoc-gen-echo-v5
    out: api
    opt:
      - paths=source_relative
      - runtime_import=github.com/ca-x/bufplugins/runtime/echoadapter
```

The Echo plugin defaults to `package_suffix=echo`, so a proto package like `helloworldv1` generates Echo registration into `helloworldv1echo`. This avoids Go import cycles with Connect's default `helloworldv1connect` package.

## Proto Usage

REST routes come from Google's standard annotations:

```proto
import "google/api/annotations.proto";

service GreeterService {
  rpc SayHello(SayHelloRequest) returns (SayHelloResponse) {
    option (google.api.http) = {
      get: "/helloworld/{name}"
    };
  }
}
```

Methods without `google.api.http` still get the Connect endpoint, but no REST route.

## Runtime Injection

Generated code exposes a DI-friendly registrar:

```go
validator := validate.MustProtovalidate()

registrar := helloworldv1echo.NewGreeterServiceEchoRegistrar(
    greeter,
    echoadapter.WithValidator(validator),
    echoadapter.WithConnectOptions(connect.WithInterceptors(...)),
)

if err := registrar.Register(e); err != nil {
    return err
}
```

The runtime keeps common behavior replaceable:

- `WithRequestBinder` and `WithMethodRequestBinder` for path/query/form/body binding.
- `WithValidator` for `buf.build/go/protovalidate`.
- `WithResponseWriter` for envelopes, redirects, headers, and custom status behavior.
- `WithErrorMapper` and `WithErrorWriter` for independent REST error policy.
- `WithMiddleware` and `WithGroupPrefix` for Echo route integration.

## Example

See [examples/echo-v5](examples/echo-v5) for a complete Buf v2 project with `google.api.http`, `buf.validate`, generated Connect code, generated Echo code, DI-style server wiring, and route tests.
