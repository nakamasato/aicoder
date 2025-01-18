package llm

import (
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
)

func generateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// GenerateJsonSchemaParam generates a JSON schema parameter for the given type.
func GenerateJsonSchemaParam[T any](name, description string) openai.ResponseFormatJSONSchemaJSONSchemaParam {
	schema := generateSchema[T]()
	return openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F(name),
		Description: openai.F(description),
		Schema:      openai.F(schema),
		Strict:      openai.Bool(true),
	}
}
