# ZOS RPC API

this package implements a jsonrpc api for all zos endpoint with `net/rpc` and `net/rpc/jsonrpc` packages.

`types.go` file is auto generated from the `openrpc.json` spec file on the repo root directory with the tool in `tools/openrpc-codegen`

to generate the types from the openrpc spec file you can use the tool manually check [docs](./../../tools/openrpc-codegen/readme.md) or use `go generate`

first you need to have:

- `openrpc-codegen` bin: `cd tools/openrpc-codegen/ && make build`
- `goimports`: `go install golang.org/x/tools/cmd/goimports@latest`

then you can generate:

`go generate ./pkg/rpc/...`

## How to call the api

it is can be called with `nc` for example:

```bash
echo '{"method": "zos.SystemVersion", "params": [], "id": 1}' | nc -q 1 <node-ip> 3000
```
