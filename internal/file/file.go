package file

import (
	"fmt"
	"strings"
)

type File struct {
	Path    string
	Content string
}

type Files []*File

// String returns a string representation of the Files.
// FilePath and Content are printed for each file.
func (f Files) String() string {
	var builder strings.Builder
	for _, file := range f {
		builder.WriteString(fmt.Sprintf("\n--------------------\nfilepath:%s\n--%s\n--- content end---", file.Path, file.Content))
	}
	return builder.String()
}
