package parser

import (
	"fmt"
	"strings"

	"github.com/kumarlokesh/sql-parser/internal/ast"
	"github.com/kumarlokesh/sql-parser/internal/lexer"
)

// Parser represents a parser.
type Parser struct {
	l *lexer.Lexer

	currentToken lexer.Token
	peekToken    lexer.Token
	errors       []string

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

type (
	prefixParseFn func() (ast.Expr, error)
	infixParseFn  func(ast.Expr) (ast.Expr, error)
)

// New creates a new Parser.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:              l,
		errors:         []string{},
		prefixParseFns: make(map[lexer.TokenType]prefixParseFn),
		infixParseFns:  make(map[lexer.TokenType]infixParseFn),
	}

	// Register prefix functions
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBoolean)
	p.registerPrefix(lexer.FALSE, p.parseBoolean)

	// Register infix functions with their precedence
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NEQ, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LTE, p.parseInfixExpression)
	p.registerInfix(lexer.GTE, p.parseInfixExpression)
	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.AND, p.parseInfixExpression)
	p.registerInfix(lexer.OR, p.parseInfixExpression)

	// Read two tokens, so currentToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken advances the parser to the next token.
func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// Parse parses the input and returns the AST.
func (p *Parser) Parse() (ast.Statement, error) {
	if p.currentToken.Type == lexer.SELECT {
		return p.parseSelectStatement()
	}

	return nil, fmt.Errorf("expected SELECT, got token type %d", p.currentToken.Type)
}

// parseSelectStatement parses a SELECT SQL statement.
func (p *Parser) parseSelectStatement() (*ast.SelectStmt, error) {
	stmt := &ast.SelectStmt{}

	// We've already seen the SELECT token, so we can proceed to parse fields
	// Parse fields
	fields, err := p.parseSelectFields()
	if err != nil {
		return nil, fmt.Errorf("error parsing fields: %v", err)
	}
	stmt.Fields = fields

	if !p.expectPeek(lexer.FROM) {
		return nil, fmt.Errorf("expected FROM, got token type %d", p.peekToken.Type)
	}
	if !p.expectPeek(lexer.IDENT) {
		return nil, fmt.Errorf("expected table name, got token type %d", p.peekToken.Type)
	}
	stmt.TableName = p.currentToken.Literal
	if p.peekTokenIs(lexer.WHERE) {
		p.nextToken() // consume WHERE

		// Ensure we're at the start of an expression
		if !p.currentTokenIs(lexer.IDENT) && !p.currentTokenIs(lexer.NUMBER) &&
			!p.currentTokenIs(lexer.STRING) && !p.currentTokenIs(lexer.TRUE) &&
			!p.currentTokenIs(lexer.FALSE) && !p.currentTokenIs(lexer.LPAREN) {
			p.nextToken() // advance to the next token if we're not at an expression start
		}

		expr, err := p.parseExpression(LOWEST)
		if err != nil {
			return nil, fmt.Errorf("error parsing WHERE clause: %v", err)
		}
		stmt.Where = expr
	}

	return stmt, nil
}

// parseSelectFields parses the list of fields in a SELECT statement.
func (p *Parser) parseSelectFields() ([]*ast.Field, error) {
	var fields []*ast.Field

	if p.peekTokenIs(lexer.ASTERISK) {
		p.nextToken()
		return []*ast.Field{{Name: "*"}}, nil
	}

	// Parse field list
	for {
		if !p.expectPeek(lexer.IDENT) {
			return nil, fmt.Errorf("expected identifier, got token type %d", p.peekToken.Type)
		}

		fields = append(fields, &ast.Field{Name: p.currentToken.Literal})

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken() // consume comma

		// If the next token is not an identifier or asterisk, we have a syntax error
		if !p.peekTokenIs(lexer.IDENT) && !p.peekTokenIs(lexer.ASTERISK) {
			return nil, fmt.Errorf("expected identifier or *, got token type %d", p.peekToken.Type)
		}
	}

	return fields, nil
}

// parseExpression parses an expression with the given precedence.
func (p *Parser) parseExpression(precedence int) (ast.Expr, error) {
	prefix := p.prefixParseFns[p.currentToken.Type]
	if prefix == nil {
		return nil, fmt.Errorf("no prefix parse function for %v found", p.currentToken.Type)
	}

	leftExp, err := prefix()
	if err != nil {
		return nil, err
	}

	for !p.peekTokenIs(lexer.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp, nil
		}

		p.nextToken()

		leftExp, err = infix(leftExp)
		if err != nil {
			return nil, err
		}
	}

	return leftExp, nil
}

// parseInfixExpression parses an infix expression.
func (p *Parser) parseInfixExpression(left ast.Expr) (ast.Expr, error) {
	expression := &ast.BinaryExpr{
		Left:  left,
		Op:    p.currentToken.Literal,
		Right: nil,
	}

	// Save the current operator precedence
	precedence := p.curPrecedence()

	// Move to the next token (the right-hand side of the operator)
	p.nextToken()

	// Parse the right-hand side with the current operator's precedence
	right, err := p.parseExpression(precedence)
	if err != nil {
		return nil, err
	}
	expression.Right = right

	return expression, nil
}

// parseIdentifier parses an identifier expression.
func (p *Parser) parseIdentifier() (ast.Expr, error) {
	return &ast.ColRef{Name: p.currentToken.Literal}, nil
}

// parseNumberLiteral parses a number literal.
func (p *Parser) parseNumberLiteral() (ast.Expr, error) {
	// Parse the string into an int64
	var val int64
	_, err := fmt.Sscanf(p.currentToken.Literal, "%d", &val)
	if err != nil {
		return nil, fmt.Errorf("could not parse %q as integer: %v", p.currentToken.Literal, err)
	}
	return &ast.NumberLit{Value: val}, nil
}

// parseStringLiteral parses a string literal.
func (p *Parser) parseStringLiteral() (ast.Expr, error) {
	// Remove the surrounding quotes
	value := p.currentToken.Literal
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
	}
	return &ast.StringLit{Value: value}, nil
}

// parseBoolean parses a boolean literal.
func (p *Parser) parseBoolean() (ast.Expr, error) {
	return &ast.BoolLit{Value: p.currentToken.Type == lexer.TRUE}, nil
}

// registerPrefix registers a prefix parser function.
func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

// registerInfix registers an infix parser function.
func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// currentTokenIs checks if the current token is of the given type.
func (p *Parser) currentTokenIs(t lexer.TokenType) bool {
	return p.currentToken.Type == t
}

// peekTokenIs checks if the next token is of the given type.
func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek checks if the next token is of the given type and advances if it is.
// Returns false and reports an error if the next token is not of the expected type.
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	// Store the error in the parser's errors slice
	err := p.peekError(t)
	if err != nil {
		p.errors = append(p.errors, err.Error())
	}
	return false
}

// peekError generates an error for an unexpected token.
// It formats a helpful error message showing what was expected vs what was found.
func (p *Parser) peekError(expected ...lexer.TokenType) error {
	if len(expected) == 0 {
		return fmt.Errorf("unexpected token %v", p.peekToken.Type)
	}

	expectedStrs := make([]string, len(expected))
	for i, t := range expected {
		expectedStrs[i] = fmt.Sprintf("%v", t)
	}

	return fmt.Errorf("expected next token to be %s, got %v instead",
		strings.Join(expectedStrs, " or "),
		p.peekToken.Type,
	)
}

const (
	_ int = iota
	LOWEST
	CONDITION // AND, OR
	EQUALS    // =, !=, <, >, <=, >=
	SUM       // +, -
	PRODUCT   // *, /
	PREFIX    // -X or !X
	CALL      // myFunction(X)
)

var precedences = map[lexer.TokenType]int{
	lexer.EQ:       EQUALS,
	lexer.NEQ:      EQUALS,
	lexer.LT:       EQUALS,
	lexer.GT:       EQUALS,
	lexer.LTE:      EQUALS,
	lexer.GTE:      EQUALS,
	lexer.AND:      CONDITION,
	lexer.OR:       CONDITION,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.SLASH:    PRODUCT,
	lexer.ASTERISK: PRODUCT,
}

// peekPrecedence returns the precedence of the next token.
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// curPrecedence returns the precedence of the current token.
func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.currentToken.Type]; ok {
		return p
	}
	return LOWEST
}
