# OpenRPC CodeGen

this tools generate server side code in go from an openrpc spec file

## Usage

Manually generate code

```bash
go run main.go -spec </path/to/specfile> -output </path/to/generated/code>
```

Use it to generate the api code

```bash
make build
# go to root
go generate
```

## Limitations

Any openrpc file that passes the linting on the [playground](https://playground.open-rpc.org/) should be valid for this tool. with just some limitations:

- Methods must have only one arg/reply: since we use `net/rpc` package it requires to have only a single arg and reply.
- Methods can have arg/reply defined only for a primitive types. but for custom types array/objects it must be defined on the components schema and referenced in the method.
- Array is not a valid reply type, you need to define an object in the components that have a field of this array. and reference it on the method.
- Method Name, component Name, and component fields name must be a `PascalCase`
- Component fields must have a tag field, it is interpreted to a json tag on the generated struct and it is necessary in the conversion to the zos types.

## Extensions

- for compatibility with gridtypes we needed to configure some extra formats like

  ```json
  "Data": {
    "tag": "data",
    "type": "object",
    "format": "raw"
  }
  ```

  which will generate a json.RawMessage type

## Notes

- All types and fields should be upper case.

## Enhancements

- [ ] write structs in order
- [ ] extend the spec file to have errors and examples and docs
