package llm

import "github.com/invopop/jsonschema"

type FileList struct {
	Paths []string `json:"paths" jsonschema_description:"Paths of the relevant files"`
}

type BlockType string

const (
	Function BlockType = "function"
	Class    BlockType = "class"
	Variable BlockType = "variable"
	Struct   BlockType = "struct"
)

type Block struct {
	BlockType BlockType `json:"block_type" jsonschema_description:"Type of the block. e.g. function, class, variable, struct etc."`
	Name      string    `json:"name" jsonschema_description:"Name of the block"`
}

type BlockWithLine struct {
	BlockType BlockType `json:"block_type" jsonschema_description:"Type of the block. e.g. function, class, variable, struct etc."`
	Name      string    `json:"name" jsonschema_description:"Name of the block"`
	Line      int       `json:"line" jsonschema_description:"Line number of the block"`
}

type BlockList struct {
	Path   string  `json:"path" jsonschema_description:"Path of the file"`
	Blocks []Block `json:"blocks" jsonschema_description:"List of blocks"`
}

type BlockWithLineList struct {
	Path   string          `json:"path" jsonschema_description:"Path of the file"`
	Blocks []BlockWithLine `json:"blocks" jsonschema_description:"List of blocks with line numbers"`
}

type FileBlockList struct {
	Files []BlockList `json:"files" jsonschema_description:"List of files with blocks"`
}

type FileBlockLineList struct {
	Files []BlockWithLineList `json:"files" jsonschema_description:"List of files with blocks and line numbers"`
}

type Repair struct {
	Change string `json:"diff" jsonschema_description:"The diff of the repair. This should start with <<<<<<< SEARCH and end with >>>>>>> REPLACE. ======= is the separator between the original and the replacement."`
}

type YesOrNo struct {
	Answer bool `json:"answer" jsonschema_description:"Answer to the yes or no question"`
}

var (
	FileListSchemaParam          = GenerateSchema[FileList]("filelist", "List of filepaths")
	FileBlockListSchemaParam     = GenerateSchema[FileBlockList]("blocklist", "List of blocks in files")
	FileBlockLineListSchemaParam = GenerateSchema[FileBlockLineList]("blocklinelist", "List of blocks with line numbers in files")
	RepairSchemaParam            = GenerateSchema[Repair]("repair", "The diff of the repair")
	YesOrNoSchemaParam           = GenerateSchema[YesOrNo]("yes_or_no", "Answer to the yes or no question")
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
		Schema:      schema,
		Name:        name,
		Description: description,
	}
}
