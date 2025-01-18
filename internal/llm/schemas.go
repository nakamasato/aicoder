package llm

type FileList struct {
	Paths []string `json:"paths" jsonschema_description:"Paths of the relevant files"`
}

type YesOrNo struct {
	Answer bool `json:"answer" jsonschema_description:"Answer to the yes or no question"`
}

var (
	FileListSchemaParam = GenerateJsonSchemaParam[FileList]("filelist", "List of filepaths")
	YesOrNoSchemaParam  = GenerateJsonSchemaParam[YesOrNo]("yes_or_no", "Answer to the yes or no question")
)
