package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "simple select",
			input: "SELECT id, name FROM users WHERE age > 18",
			expected: []Token{
				{Type: SELECT, Literal: "SELECT", Pos: Position{Line: 1, Column: 1}},
				{Type: IDENT, Literal: "id", Pos: Position{Line: 1, Column: 8}},
				{Type: COMMA, Literal: ",", Pos: Position{Line: 1, Column: 10}},
				{Type: IDENT, Literal: "name", Pos: Position{Line: 1, Column: 12}},
				{Type: FROM, Literal: "FROM", Pos: Position{Line: 1, Column: 17}},
				{Type: IDENT, Literal: "users", Pos: Position{Line: 1, Column: 22}},
				{Type: WHERE, Literal: "WHERE", Pos: Position{Line: 1, Column: 28}},
				{Type: IDENT, Literal: "age", Pos: Position{Line: 1, Column: 34}},
				{Type: GT, Literal: ">", Pos: Position{Line: 1, Column: 38}},
				{Type: NUMBER, Literal: "18", Pos: Position{Line: 1, Column: 40}},
				{Type: EOF, Literal: "", Pos: Position{Line: 1, Column: 42}},
			},
		},
		{
			name:  "string literals",
			input: "SELECT * FROM users WHERE name = 'John''s name' AND active = true",
			expected: []Token{
				{Type: SELECT, Literal: "SELECT", Pos: Position{Line: 1, Column: 1}},
				{Type: ASTERISK, Literal: "*", Pos: Position{Line: 1, Column: 8}},
				{Type: FROM, Literal: "FROM", Pos: Position{Line: 1, Column: 10}},
				{Type: IDENT, Literal: "users", Pos: Position{Line: 1, Column: 15}},
				{Type: WHERE, Literal: "WHERE", Pos: Position{Line: 1, Column: 21}},
				{Type: IDENT, Literal: "name", Pos: Position{Line: 1, Column: 27}},
				{Type: EQ, Literal: "=", Pos: Position{Line: 1, Column: 32}},
				{Type: STRING, Literal: "John's name", Pos: Position{Line: 1, Column: 34}},
				{Type: AND, Literal: "AND", Pos: Position{Line: 1, Column: 49}},
				{Type: IDENT, Literal: "active", Pos: Position{Line: 1, Column: 53}},
				{Type: EQ, Literal: "=", Pos: Position{Line: 1, Column: 60}},
				{Type: TRUE, Literal: "true", Pos: Position{Line: 1, Column: 62}},
				{Type: EOF, Literal: "", Pos: Position{Line: 1, Column: 66}},
			},
		},
		{
			name:  "numeric literals",
			input: "SELECT 42, 3.14, -1, 1e10, 2.5e-3",
			expected: []Token{
				{Type: SELECT, Literal: "SELECT", Pos: Position{Line: 1, Column: 1}},
				{Type: NUMBER, Literal: "42", Pos: Position{Line: 1, Column: 8}},
				{Type: COMMA, Literal: ",", Pos: Position{Line: 1, Column: 10}},
				{Type: NUMBER, Literal: "3.14", Pos: Position{Line: 1, Column: 12}},
				{Type: COMMA, Literal: ",", Pos: Position{Line: 1, Column: 16}},
				{Type: MINUS, Literal: "-", Pos: Position{Line: 1, Column: 18}},
				{Type: NUMBER, Literal: "1", Pos: Position{Line: 1, Column: 19}},
				{Type: COMMA, Literal: ",", Pos: Position{Line: 1, Column: 20}},
				{Type: NUMBER, Literal: "1e10", Pos: Position{Line: 1, Column: 22}},
				{Type: COMMA, Literal: ",", Pos: Position{Line: 1, Column: 26}},
				{Type: NUMBER, Literal: "2.5e-3", Pos: Position{Line: 1, Column: 28}},
				{Type: EOF, Literal: "", Pos: Position{Line: 1, Column: 34}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)

			for i, expectedToken := range tt.expected {
				tok := l.NextToken()

				if tok.Type != expectedToken.Type {
					t.Fatalf("tests[%d] - token type wrong. expected=%q, got=%q",
						i, expectedToken.Type, tok.Type)
				}

				if tok.Literal != expectedToken.Literal {
					t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
						i, expectedToken.Literal, tok.Literal)
				}

				if tok.Pos.Line != expectedToken.Pos.Line ||
					tok.Pos.Column != expectedToken.Pos.Column {
					t.Fatalf("tests[%d] - position wrong. expected=%+v, got=%+v",
						i, expectedToken.Pos, tok.Pos)
				}
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid query",
			input:   "SELECT * FROM users",
			wantErr: false,
		},
		{
			name:    "invalid character",
			input:   "SELECT @ FROM users",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Tokenize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Tokenize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
