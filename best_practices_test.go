package containerhive_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestContextParameterBestPractices checks that functions accepting context.Context
// as their first parameter follow Go best practices
func TestContextParameterBestPractices(t *testing.T) {
	// Parse all Go files in the project
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}

	issuesFound := false

	// Check each package
	for _, pkg := range pkgs {
		// Check each file in the package
		for _, file := range pkg.Files {
			// Inspect all function declarations
			ast.Inspect(file, func(n ast.Node) bool {
				// Check if it's a function declaration
				if fn, ok := n.(*ast.FuncDecl); ok {
					// Skip test functions and main functions
					if fn.Name.Name == "main" || fn.Name.Name == "TestMain" {
						return true
					}

					// Check if function has parameters
					if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
						// Get the first parameter
						firstParam := fn.Type.Params.List[0]

						// Check if it's a context.Context parameter
						if isContextParam(firstParam) {
							// Check if context parameter is named "ctx"
							if !isNamedCtx(firstParam) {
								t.Logf("Function %s should name its context parameter 'ctx' (found: %s)",
									fn.Name.Name, getParamName(firstParam))
								issuesFound = true
							}

							// Check if context parameter is the first parameter
							// (it is, by our check above)

							// Check if function returns error as last return value
							if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
								lastResult := fn.Type.Results.List[len(fn.Type.Results.List)-1]
								if !isErrorType(lastResult.Type) {
									t.Logf("Function %s with context parameter should return error as last value",
										fn.Name.Name)
									issuesFound = true
								}
							}
						}
					}
				}
				return true
			})
		}
	}

	if issuesFound {
		t.Log("Context parameter best practices issues found. See logs above.")
	} else {
		t.Log("All context parameter usage follows best practices.")
	}
}

// isContextParam checks if a parameter is of type context.Context
func isContextParam(param *ast.Field) bool {
	// Check if it's a selector expression like "context.Context"
	if selExpr, ok := param.Type.(*ast.SelectorExpr); ok {
		if selExpr.X.(*ast.Ident).Name == "context" && selExpr.Sel.Name == "Context" {
			return true
		}
	}
	// Check if it's an identifier "Context" (assuming context is imported)
	if ident, ok := param.Type.(*ast.Ident); ok {
		if ident.Name == "Context" {
			return true
		}
	}
	return false
}

// isNamedCtx checks if a parameter is named "ctx"
func isNamedCtx(param *ast.Field) bool {
	if len(param.Names) == 0 {
		return false // unnamed parameter
	}
	return param.Names[0].Name == "ctx"
}

// getParamName gets the name of a parameter
func getParamName(param *ast.Field) string {
	if len(param.Names) == 0 {
		return "_" // unnamed parameter
	}
	return param.Names[0].Name
}

// isErrorType checks if a type is error
func isErrorType(expr ast.Expr) bool {
	// Check if it's an identifier "error"
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "error"
	}
	// Check if it's a selector expression (less common for error)
	if selExpr, ok := expr.(*ast.SelectorExpr); ok {
		return selExpr.Sel.Name == "error"
	}
	return false
}
