package parser

import (
	"fmt"
	"testing"

	"github.com/kumarlokesh/sql-parser/internal/ast"
	"github.com/kumarlokesh/sql-parser/internal/lexer"
)

func debugPrintAST(expr ast.Expr, indent string) string {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		return fmt.Sprintf("%sBinaryExpr{\n%s  Op: %q,\n%s  Left: %s,\n%s  Right: %s\n%s}",
			indent, indent, e.Op, indent, debugPrintAST(e.Left, indent+"  "), indent, debugPrintAST(e.Right, indent+"  "), indent)
	case *ast.ColRef:
		return fmt.Sprintf("%sColRef{Name: %q}", indent, e.Name)
	case *ast.NumberLit:
		return fmt.Sprintf("%sNumberLit{Value: %d}", indent, e.Value)
	case *ast.StringLit:
		return fmt.Sprintf("%sStringLit{Value: %q}", indent, e.Value)
	case *ast.BoolLit:
		return fmt.Sprintf("%sBoolLit{Value: %v}", indent, e.Value)
	default:
		return fmt.Sprintf("%s%T{}", indent, expr)
	}
}

func TestSelectStatement(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ast.SelectStmt
		wantErr bool
	}{
		{
			name:  "simple select",
			input: "SELECT id, name FROM users",
			want: &ast.SelectStmt{
				Fields: []*ast.Field{
					{Name: "id"},
					{Name: "name"},
				},
				TableName: "users",
			},
			wantErr: false,
		},
		{
			name:  "select with where clause",
			input: "SELECT * FROM users WHERE age > 18",
			want: &ast.SelectStmt{
				Fields:    []*ast.Field{{Name: "*"}},
				TableName: "users",
				Where: &ast.BinaryExpr{
					Left:  &ast.ColRef{Name: "age"},
					Op:    ">",
					Right: &ast.NumberLit{Value: 18},
				},
			},
			wantErr: false,
		},
		{
			name:  "select with string comparison",
			input: "SELECT name FROM users WHERE name = 'John' AND active = true",
			want: &ast.SelectStmt{
				Fields:    []*ast.Field{{Name: "name"}},
				TableName: "users",
				Where: &ast.BinaryExpr{
					Left: &ast.BinaryExpr{
						Left:  &ast.ColRef{Name: "name"},
						Op:    "=",
						Right: &ast.StringLit{Value: "John"},
					},
					Op: "AND",
					Right: &ast.BinaryExpr{
						Left:  &ast.ColRef{Name: "active"},
						Op:    "=",
						Right: &ast.BoolLit{Value: true},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			got, err := p.Parse()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			stmt, ok := got.(*ast.SelectStmt)
			if !ok {
				t.Fatalf("Parse() = %T, want *ast.SelectStmt", got)
			}

			if len(stmt.Fields) != len(tt.want.Fields) {
				t.Fatalf("got %d fields, want %d", len(stmt.Fields), len(tt.want.Fields))
			}

			for i, f := range stmt.Fields {
				if f.Name != tt.want.Fields[i].Name {
					t.Errorf("field[%d] = %q, want %q", i, f.Name, tt.want.Fields[i].Name)
				}
			}

			if stmt.TableName != tt.want.TableName {
				t.Errorf("table name = %q, want %q", stmt.TableName, tt.want.TableName)
			}

			if tt.want.Where != nil {
				if stmt.Where == nil {
					t.Error("expected where clause, got none")
				} else if !compareExpr(stmt.Where, tt.want.Where) {
					t.Errorf("where clause mismatch\ngot: %s\nwant: %s",
						debugPrintAST(stmt.Where, "  "),
						debugPrintAST(tt.want.Where, "  "))
				}
			} else if stmt.Where != nil {
				t.Errorf("unexpected where clause: %s", debugPrintAST(stmt.Where, "  "))
			}
		})
	}
}

func compareExpr(a, b ast.Expr) bool {
	switch a := a.(type) {
	case *ast.BinaryExpr:
		b, ok := b.(*ast.BinaryExpr)
		if !ok {
			return false
		}
		return compareExpr(a.Left, b.Left) && a.Op == b.Op && compareExpr(a.Right, b.Right)
	case *ast.ColRef:
		b, ok := b.(*ast.ColRef)
		if !ok {
			return false
		}
		return a.Name == b.Name
	case *ast.NumberLit:
		b, ok := b.(*ast.NumberLit)
		if !ok {
			return false
		}
		return a.Value == b.Value
	case *ast.StringLit:
		b, ok := b.(*ast.StringLit)
		if !ok {
			return false
		}
		return a.Value == b.Value
	case *ast.BoolLit:
		b, ok := b.(*ast.BoolLit)
		if !ok {
			return false
		}
		return a.Value == b.Value
	default:
		return false
	}
}
