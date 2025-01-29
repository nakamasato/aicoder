package llm

import "github.com/invopop/jsonschema"

type FileList struct {
	Paths []string `json:"paths" jsonschema_description:"Paths of the relevant files"`
}

type YesOrNo struct {
	Answer bool `json:"answer" jsonschema_description:"Answer to the yes or no question"`
}

var (
	FileListSchemaParam = GenerateSchema[FileList]("filelist", "List of filepaths")
	YesOrNoSchemaParam  = GenerateSchema[YesOrNo]("yes_or_no", "Answer to the yes or no question")
)

type Schema struct {
	Name        string
	Description string
	Schema      interface{}
}

func GenerateSchema[T any](name, description string) Schema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return Schema{
		Schema: schema,
		Name: name,
		Description: description,
	}
}
