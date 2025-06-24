package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kumarlokesh/sql-parser/internal/ast"
	"github.com/kumarlokesh/sql-parser/internal/lexer"
	"github.com/kumarlokesh/sql-parser/internal/parser"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: sql-parser \"SELECT * FROM table WHERE condition\"")
	}

	query := os.Args[1]
	fmt.Printf("Parsing query: %s\n\n", query)

	l := lexer.New(query)
	p := parser.New(l)

	stmt, err := p.Parse()
	if err != nil {
		log.Fatalf("Error parsing query: %v", err)
	}

	printStatement(stmt)
}

// printStatement prints the parsed statement in a readable format
func printStatement(node ast.Node) {
	switch stmt := node.(type) {
	case *ast.SelectStmt:
		fmt.Println("SELECT")
		fmt.Println("  Fields:")
		for _, field := range stmt.Fields {
			if field.Name == "*" {
				fmt.Println("    * (all columns)")
			} else {
				fmt.Printf("    %s\n", field.Name)
			}
		}

		fmt.Printf("  From: %s\n", stmt.TableName)

		if stmt.Where != nil {
			fmt.Println("  Where:")
			printExpression(stmt.Where, "    ")
		}

	default:
		fmt.Println("Unsupported statement type")
	}
}

// printExpression recursively prints an expression
func printExpression(expr ast.Expr, indent string) {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		fmt.Printf("%sBinary Expression: %s\n", indent, e.Op)
		fmt.Printf("%s  Left:\n", indent)
		printExpression(e.Left, indent+"    ")
		fmt.Printf("%s  Right:\n", indent)
		printExpression(e.Right, indent+"    ")
	case *ast.ColRef:
		fmt.Printf("%sColumn: %s\n", indent, e.Name)
	case *ast.NumberLit:
		fmt.Printf("%sNumber: %d\n", indent, e.Value)
	case *ast.StringLit:
		fmt.Printf("%sString: '%s'\n", indent, e.Value)
	case *ast.BoolLit:
		fmt.Printf("%sBoolean: %t\n", indent, e.Value)
	default:
		fmt.Printf("%sUnknown expression type: %T\n", indent, e)
	}
}
