package debug

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"

	"github.com/spf13/cobra"
)

func parseCommand() *cobra.Command {
	parseCmd := &cobra.Command{
		Use:   "ast",
		Short: "Refactor the code with AST (WIP)",
		Run:   runParse,
	}
	parseCmd.Flags().StringVarP(&filename, "filename", "f", "", "File to refactor")
	return parseCmd
}

func runParse(cmd *cobra.Command, args []string) {
	fset := token.NewFileSet()
	outputFile := fmt.Sprintf("%s.tmp", filename)
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.Name == "oldFunctionName" {
				fn.Name.Name = "newFunctionName"
			}
		}
		return true
	})

	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	err = printer.Fprint(outFile, fset, node)
	if err != nil {
		log.Fatal(err)
	}
}
