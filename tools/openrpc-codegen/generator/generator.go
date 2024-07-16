package generator

import (
	"bytes"
	"fmt"

	"github.com/threefoldtech/zos/tools/openrpc-codegen/schema"
)

func generateStructs(buf *bytes.Buffer, name string, schema schema.Schema) error {
	fields := []field{}
	for n, prop := range schema.Properties {
		// Handle nested properties recursively
		propType, err := generateType(buf, n, prop)
		if err != nil {
			return err
		}
		fields = append(fields, field{
			Name:     n,
			Type:     propType,
			JSONName: prop.Tag,
		})
	}

	return executeTemplate(buf, structTemplate, structType{
		Name:   name,
		Fields: fields,
	})
}

func generateType(buf *bytes.Buffer, name string, schema schema.Schema) (string, error) {
	if schema.Ref != "" {
		return invokeRef(schema.Ref), nil
	}

	if schema.Format == "raw" {
		return convertToGoType(invokeRef(schema.Type), schema.Format), nil
	}

	switch schema.Type {
	case "object":
		if err := generateStructs(buf, name, schema); err != nil {
			return "", err
		}
		return name, nil
	case "array":
		if schema.Items.Ref != "" {
			return "[]" + invokeRef(schema.Items.Ref), nil
		}
		itemType, err := generateType(buf, name, *schema.Items)
		if err != nil {
			return "", err
		}
		return "[]" + itemType, nil
	default:
		return convertToGoType(invokeRef(schema.Type), schema.Format), nil
	}
}

func generateSchemas(buf *bytes.Buffer, schemas map[string]schema.Schema) error {
	for key, schema := range schemas {
		_, err := generateType(buf, key, schema)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateMethods(buf *bytes.Buffer, serviceName string, methods []schema.Method) error {
	ms := []methodType{}
	for _, method := range methods {
		methodName := extractMethodName(method.Name)
		arg, reply, err := getMethodTypes(method)
		if err != nil {
			return err
		}
		ms = append(ms, methodType{
			Name:      methodName,
			ArgType:   arg,
			ReplyType: reply,
		})
	}

	return executeTemplate(buf, MethodTemplate, service{
		Name:    serviceName,
		Methods: ms,
	})
}

func getMethodTypes(method schema.Method) (string, string, error) {
	argType, replyType := "any", ""

	if len(method.Params) == 1 {
		argType = getTypeFromSchema(method.Params[0].Schema)
	} else if len(method.Params) > 1 {
		return "", "", fmt.Errorf("multiple parameters not supported for method: %v", method.Name)
	}

	if method.Result.Schema.Type != "" {
		replyType = convertToGoType(method.Result.Schema.Type, method.Result.Schema.Format)
	} else if method.Result.Schema.Ref != "" {
		replyType = invokeRef(method.Result.Schema.Ref)
	} else {
		return "", "", fmt.Errorf("no result defined for method: %v", method.Name)
	}

	return argType, replyType, nil
}

func getTypeFromSchema(schema schema.Schema) string {
	if schema.Type != "" {
		return convertToGoType(schema.Type, schema.Format)
	}
	return invokeRef(schema.Ref)
}

func GenerateServerCode(buf *bytes.Buffer, spec schema.Spec, pkg string) error {
	if err := addPackageName(buf, pkg); err != nil {
		return fmt.Errorf("failed to write pkg name: %w", err)
	}

	if err := generateMethods(buf, spec.Info.Title, spec.Methods); err != nil {
		return fmt.Errorf("failed to generate methods: %w", err)
	}

	if err := generateSchemas(buf, spec.Components.Schemas); err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	return nil
}
