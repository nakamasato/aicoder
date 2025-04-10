package locator

import (
	"testing"

	"github.com/nakamasato/aicoder/internal/loader"
)

func TestMakePrompt(t *testing.T) {
	tests := []struct {
		name          string
		templatefile  string
		query         string
		repoStructure loader.RepoStructure
		wantErr       bool
	}{
		{
			name:          "Relevant template",
			templatefile:  promptLocateFileTemplate,
			query:         "test query",
			repoStructure: loader.RepoStructure{
				// Add mock data for RepoStructure
			},
			wantErr: false,
		},
		{
			name:          "Irrelevant template",
			templatefile:  promptLocateFileIrrelevantTemplate,
			query:         "test query",
			repoStructure: loader.RepoStructure{},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeLocateFilePrompt(tt.templatefile, tt.query, tt.repoStructure)
			if (err != nil) != tt.wantErr {
				t.Errorf("makePrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Add additional checks for the content of `got` if necessary
				if got == "" {
					t.Errorf("makePrompt() got empty string, expected non-empty")
				}
			}
		})
	}
}
