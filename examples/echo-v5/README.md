# Echo v5 Buf Example

This example is a Buf v2 project for testing `protoc-gen-echo-v5`.

It generates Go protobuf types, Connect handlers, and Echo v5 REST registration code into `api` using `paths=source_relative`. The proto uses `google.api.http` annotations for REST routes and `buf.validate` annotations for request validation through `buf.build/go/protovalidate`.

Generated packages:

```text
api/helloworld/v1                         # protobuf messages
api/helloworld/v1/helloworldv1connect     # connect-go handlers
api/helloworld/v1/helloworldv1echo        # Echo v5 registrar
```

## Generate

From this directory:

```sh
go install ../../cmd/protoc-gen-echo-v5
buf dep update
buf generate
```

`buf.gen.yaml` expects `protoc-gen-echo-v5` to be available on `PATH`.

## Test

After generated files exist:

```sh
go mod tidy
go test ./...
```

The tests cover Echo REST routing, validation errors, Connect route mounting, and DI-friendly registrar construction.

## Routes

- `GET /helloworld/{name}` maps to `GreeterService.SayHello`.
- `GET /search/{keyword}` maps to `GreeterService.LuckySearch`.
- `POST /greetings` maps to `GreeterService.CreateGreeting`, binds the full JSON body, and returns the `message` response field.

## Server Wiring

`internal/server` constructs the generated registrar directly:

```go
validator, err := validate.NewProtovalidate()
if err != nil {
    return err
}

registrar := helloworldv1echo.NewGreeterServiceEchoRegistrar(
    greeter,
    echoadapter.WithValidator(validator),
)

return registrar.Register(e)
```
