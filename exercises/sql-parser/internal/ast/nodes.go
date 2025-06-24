package ast

// Node represents a node in the Abstract Syntax Tree (AST).
type Node interface {
	// node is an unexported method to ensure only types in this package
	// can be AST nodes.
	node()
}

// Statement represents a SQL statement.
type Statement interface {
	Node
	stmt()
}

// SelectStmt represents a SELECT SQL statement.
type SelectStmt struct {
	// Fields is the list of columns being selected.
	Fields []*Field
	// TableName is the name of the table to select from.
	TableName string
	// Where is the WHERE clause expression, if any.
	Where Expr
}

// node implements the Node interface.
func (s *SelectStmt) node() {}

// stmt implements the Statement interface.
func (s *SelectStmt) stmt() {}

// Field represents a selected field in a SELECT statement.
type Field struct {
	// Name is the name of the field.
	Name string
}

// Expr represents an expression in SQL.
type Expr interface {
	Node
	expr()
}

// BinaryExpr represents a binary expression (e.g., age > 18).
type BinaryExpr struct {
	// Left is the left-hand side of the expression.
	Left Expr
	// Op is the operator (e.g., "=", "!=", ">", "<", etc.).
	Op string
	// Right is the right-hand side of the expression.
	Right Expr
}

func (b *BinaryExpr) node() {}
func (b *BinaryExpr) expr() {}

// ColRef represents a column reference (e.g., users.id).
type ColRef struct {
	// Name is the name of the column.
	Name string
}

func (c *ColRef) node() {}
func (c *ColRef) expr() {}

// NumberLit represents a numeric literal (e.g., 42).
type NumberLit struct {
	// Value is the numeric value.
	Value int64
}

func (n *NumberLit) node() {}
func (n *NumberLit) expr() {}

// StringLit represents a string literal (e.g., 'hello').
type StringLit struct {
	// Value is the string value, without surrounding quotes.
	Value string
}

func (s *StringLit) node() {}
func (s *StringLit) expr() {}

// BoolLit represents a boolean literal (true or false).
type BoolLit struct {
	// Value is the boolean value.
	Value bool
}

func (b *BoolLit) node() {}
func (b *BoolLit) expr() {}
