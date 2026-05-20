# Echo v5 Plugin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `protoc-gen-echo-v5`, reusable runtime packages, and a complete `examples/echo-v5` Buf project.

**Architecture:** `cmd/` contains the plugin binary, `internal/generator` contains protogen logic, `runtime/httpadapter` contains framework-neutral contracts, and `runtime/echoadapter` adapts those contracts to Echo v5. Generated code exposes DI-friendly registrar structs plus convenience registration functions.

**Tech Stack:** Go 1.25+, `google.golang.org/protobuf/compiler/protogen`, `connectrpc.com/connect`, `github.com/labstack/echo/v5`, `google.api.http`, and `buf.build/go/protovalidate`.

---

### Task 1: Module And Runtime Contracts

**Files:**
- Create: `go.mod`
- Create: `runtime/httpadapter/spec.go`
- Create: `runtime/httpadapter/binding/binder.go`
- Create: `runtime/httpadapter/response/writer.go`
- Create: `runtime/httpadapter/errors/errors.go`
- Create: `runtime/httpadapter/validate/validate.go`
- Create: `runtime/echoadapter/options.go`
- Create: `runtime/echoadapter/register.go`

- [ ] Define framework-neutral specs for services, methods, HTTP bindings, path variables, and request/response constructors.
- [ ] Define injectable binder, validator, response writer, error mapper, and error writer interfaces.
- [ ] Implement default Echo options, per-method override maps, and route registration helpers.
- [ ] Add a protovalidate adapter backed by `buf.build/go/protovalidate`.

### Task 2: Generator

**Files:**
- Create: `cmd/protoc-gen-echo-v5/main.go`
- Create: `internal/generator/options.go`
- Create: `internal/generator/google_http.go`
- Create: `internal/generator/echo_v5.go`

- [ ] Parse plugin options: `runtime_import`, `file_suffix`, and `connect_package_suffix`.
- [ ] Use `protogen` native handling for `paths` and `module`.
- [ ] Read `google.api.http` extensions from method options.
- [ ] Expand `additional_bindings`.
- [ ] Convert Google path templates to Echo v5 path syntax.
- [ ] Generate registrar structs, convenience functions, Connect route registration, and REST unary route registration.

### Task 3: Example Project

**Files:**
- Create: `examples/echo-v5/buf.yaml`
- Create: `examples/echo-v5/buf.gen.yaml`
- Create: `examples/echo-v5/go.mod`
- Create: `examples/echo-v5/proto/helloworld/v1/helloworld.proto`
- Create: `examples/echo-v5/internal/server/server.go`
- Create: `examples/echo-v5/internal/server/server_test.go`
- Create: `examples/echo-v5/cmd/server/main.go`
- Create: `examples/echo-v5/README.md`

- [ ] Use Buf v2 managed mode and clean plugin list: Go, Connect-Go, local Echo plugin.
- [ ] Use `google.api.http` for REST routes and `buf.validate` for validation rules.
- [ ] Show DI-friendly registrar construction in the server package.
- [ ] Test REST route behavior, validation errors, and Connect route behavior.

### Task 4: Integrated Verification

**Files:**
- Modify generated files under `examples/echo-v5/api` through `buf generate`.

- [ ] Run `go mod tidy`.
- [ ] Run `go test ./...`.
- [ ] Run `go build ./cmd/protoc-gen-echo-v5`.
- [ ] Run `buf generate` in `examples/echo-v5`.
- [ ] Run `go mod tidy` in `examples/echo-v5`.
- [ ] Run `go test ./...` in `examples/echo-v5`.
