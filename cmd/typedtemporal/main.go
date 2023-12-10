package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"unicode"
)

var debug = flag.Bool("debug", false, "Enables debugging compile errors")

func main() {
	flag.Parse()
	if err := run(*debug); err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}
}

const templateText = `package {{.PackageName}}

import (
	"context"

	"go.temporal.io/sdk/client"
)

{{ range .Data }}

func Execute{{ UppercaseFirst .WorkflowName }}Workflow(ctx context.Context,
	c client.Client,
	options client.StartWorkflowOptions,
	input *{{ FormatString .InputStructName }}) ({{ FormatString .WorkflowName }}Run, error) {
	r, err := c.ExecuteWorkflow(ctx, options, {{ FormatString .WorkflowFunc }}, input)
	if err != nil {
		return nil, err
	}

	return &{{ FormatString .WorkflowName }}RunImpl{
		c:   c,
		run: r,
	}, nil

}

type temporal{{ FormatString .WorkflowName }}WorkflowRun interface {
	Get(context.Context, interface{}) error
}

type {{ FormatString .WorkflowName }}Run interface {
	Get(context.Context, *{{ .ResultStructName }}) error
}

type {{ FormatString .WorkflowName }}RunImpl struct {
	c   client.Client
	run temporal{{ FormatString .WorkflowName }}WorkflowRun
}

func (s *{{ FormatString .WorkflowName }}RunImpl) Get(ctx context.Context, result *{{ FormatString .ResultStructName }}) error {
	return s.run.Get(ctx, result)
}

{{ end }}
`

type templateData struct {
	PackageName string
	Data        []workflowData
}

type workflowData struct {
	WorkflowName     string
	WorkflowFunc     string
	ResultStructName string
	InputStructName  string
}

func run(debug bool) error {
	outputFile := "workflow_results_gen.go"
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	workflowsFile := os.Getenv("GOFILE")
	fmt.Printf("Parsing %s/%s...\n", pwd, workflowsFile)

	fset := token.NewFileSet()

	// Parse the source file
	file, err := parser.ParseFile(fset, path.Join(pwd, workflowsFile), nil, parser.ParseComments)
	if err != nil {
		fmt.Println("Error parsing file:", err)
		os.Exit(1)
	}

	parsedData := templateData{PackageName: file.Name.Name}

	// Look for the workflows variable
	var workflowsExpr ast.Expr
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					vs := valueSpec.Values[0]
					if r, ok := vs.(*ast.CompositeLit); ok {
						if arr, ok := r.Type.(*ast.ArrayType); ok {
							if _, ok := arr.Elt.(*ast.Ident); ok {
								workflowsExpr = r
							} else if elt2, ok := arr.Elt.(*ast.SelectorExpr); ok {
								fmt.Printf("found object of correct type: %s at line %d\n", elt2.Sel.Name, elt2.Pos())
								workflowsExpr = r
								break
							}

						}
					}
				}
			}
		}
	}

	// Check if workflowsExpr is a slice literal
	if sliceExpr, ok := workflowsExpr.(*ast.CompositeLit); ok {
		// Prepare data for the template
		var data []workflowData
		for _, elt := range sliceExpr.Elts {
			if arrElems, ok := elt.(*ast.CompositeLit); ok {
				fmt.Println("______ START OF WORKFLOW OBJ ______")
				dat := workflowData{}
				found := false
				for _, arrElt := range arrElems.Elts {

					if arrElem, ok := arrElt.(*ast.KeyValueExpr); ok {
						fmt.Println("elem key", arrElem.Key)

						if key, ok := arrElem.Key.(*ast.Ident); ok {
							if v, ok := arrElem.Value.(*ast.BasicLit); ok {
								switch key.Name {
								case "Name":
									fmt.Println("Found workflow name value")
									dat.WorkflowName = v.Value
								case "Description":
									fmt.Println("TODO: use description")
								default:
									fmt.Println("VAL WAS", key.Name)
								}
							}
							if key.Name == "Func" {
								fmt.Println("found the workflow func")
								found = true
								switch eltTyped := arrElem.Value.(type) {
								case *ast.FuncLit:
									fmt.Println("literal")
									dat.InputStructName = getInputStructName(eltTyped.Type, 1, 2)
									dat.ResultStructName = getResultStructName(eltTyped.Type, 0, 2)
									// dat.WorkflowFunc = getStructNameFromFunction(eltTyped) TODO inline functions

								case *ast.Ident:
									fmt.Println("named func")
									// Check if it's a named function
									if funcDecl, ok := getFunctionDeclaration(file, eltTyped.Name); ok {
										dat.InputStructName = getInputStructName(funcDecl.Type, 1, 2)
										dat.ResultStructName = getResultStructName(funcDecl.Type, 0, 2)
										dat.WorkflowFunc = funcDecl.Name.Name
									} else {
										fmt.Println("NOT OK FUNC DECLARATION")
									}
								default:
									fmt.Println("type not found: ", eltTyped)
								}
							}
						}
					}
				}
				if found {
					data = append(data, dat)

				}
				fmt.Println("dat: ", dat.WorkflowName, dat.InputStructName, dat.ResultStructName, dat.WorkflowFunc)
				// Check if the element is a function literal
			}

		}
		parsedData.Data = data
		fmt.Println(data)

		// Open a file for writing the generated code
		outputFilePath := filepath.Join(filepath.Dir(workflowsFile), outputFile)
		outputFile, err := os.Create(outputFilePath)
		if err != nil {
			fmt.Println("Error creating output file:", err)
			os.Exit(1)
		}
		defer outputFile.Close()
		var tmpl = template.Must(template.New("result").Funcs(template.FuncMap{"FormatString": FormatString, "LowerFirst": LowercaseFirst, "UppercaseFirst": UppercaseFirst}).Parse(templateText))

		// Execute the template and write the result to the output file
		err = tmpl.Execute(outputFile, parsedData)
		if err != nil {
			fmt.Println("Error executing template:", err)
			os.Exit(1)
		}
	}
	return nil
}

// FormatString formats a string as a quoted string
func FormatString(s string) string {
	return strings.Trim(s, "\"")
}

// UppercaseFirst uppercases the first character of a string
func UppercaseFirst(s string) string {
	s = strings.Trim(s, "\"")
	if s == "" {
		return s
	}
	fmt.Println(string(unicode.ToUpper(rune(s[0]))))
	return string(unicode.ToUpper(rune(s[0]))) + s[1:]
}

// UppercaseFirst uppercases the first character of a string
func LowercaseFirst(s string) string {
	if s == "" {
		return s
	}
	return string(unicode.ToLower(rune(s[0]))) + s[1:]
}

func getInputStructName(funcType *ast.FuncType, argIndex int, expectedInputsLength int) string {
	if expectedInputsLength < 1 || argIndex > expectedInputsLength {
		fmt.Println("Unexpected param list. Total Length: ", expectedInputsLength)
		os.Exit(1)
	}

	if len(funcType.Params.List) != expectedInputsLength {
		fmt.Println("Function had unexpected number of inputs")
		fmt.Println("Expected:  ", expectedInputsLength, ". Got: ", len(funcType.Params.List))
		os.Exit(1)
	}
	structType, ok := funcType.Params.List[argIndex].Type.(*ast.Ident)
	if !ok {
		fmt.Println("Error: Input Parameter type must be a struct.", structType, funcType.Params.List[argIndex].Type)
		os.Exit(1)
	}
	return structType.Name
}

func getResultStructName(funcType *ast.FuncType, argIndex int, expectedReturnLength int) string {
	if expectedReturnLength < 1 || argIndex > expectedReturnLength {
		fmt.Println("Unexpected param list. Total Length: ", expectedReturnLength)
		os.Exit(1)
	}

	if len(funcType.Results.List) != expectedReturnLength {
		fmt.Println("Function had unexpected number of inputs")
		fmt.Println("Expected:  ", expectedReturnLength, ". Got: ", len(funcType.Params.List))
		os.Exit(1)
	}
	structType, ok := funcType.Results.List[argIndex].Type.(*ast.Ident)
	if !ok {
		fmt.Println("Error: Result Parameter type must be a struct.", structType, funcType.Results.List[argIndex].Type)
		printUnderlyingType(funcType.Results.List[argIndex].Type)
		os.Exit(1)
	}
	return structType.Name
}

// func getStructNameFromFunction(funcDecl *ast.FuncDecl) string {

// 	n, ok := funcDecl.Name
// 	return structType.Name
// 	// Assuming the struct type has a Name (the type name)
// }

func getFunctionDeclaration(file *ast.File, functionName string) (*ast.FuncDecl, bool) {
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Name.Name == functionName {
			return funcDecl, true
		}
	}
	fmt.Println("FUNCTION: ", functionName, "not found in: ", file.Decls[0])
	return nil, false
}

func printUnderlyingType(expr ast.Expr) {
	// Use reflection to get the underlying type
	underlyingType := reflect.TypeOf(expr).Elem()

	fmt.Println("Underlying Type:", underlyingType)
}
