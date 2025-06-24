package lexer

import "strings"

// TokenType represents the type of a token.
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	WS // Whitespace

	// Literals
	IDENT  // table_name, column_name
	NUMBER // 123, 3.14
	STRING // 'hello'

	// Operators
	EQ       // =
	NEQ      // != or <>
	LT       // <
	GT       // >
	LTE      // <=
	GTE      // >=
	PLUS     // +
	MINUS    // -
	ASTERISK // *
	SLASH    // /

	// Delimiters
	COMMA     // ,
	SEMICOLON // ;
	LPAREN    // (
	RPAREN    // )

	// Keywords
	SELECT
	FROM
	WHERE
	AND
	OR
	NOT
	TRUE
	FALSE
	NULL
)

var keywords = map[string]TokenType{
	"SELECT": SELECT,
	"FROM":   FROM,
	"WHERE":  WHERE,
	"AND":    AND,
	"OR":     OR,
	"NOT":    NOT,
	"TRUE":   TRUE,
	"FALSE":  FALSE,
	"NULL":   NULL,
}

// Token represents a token or text string returned from the scanner.
type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}

// Position represents the position of a token in the input string.
type Position struct {
	Line   int
	Column int
}

// LookupIdent checks if the identifier is a keyword and returns the appropriate token type.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[strings.ToUpper(ident)]; ok {
		return tok
	}
	return IDENT
}
