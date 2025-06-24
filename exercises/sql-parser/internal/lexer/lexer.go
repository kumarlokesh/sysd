package lexer

import (
	"fmt"
	"strings"
)

// Lexer holds the state of the scanner.
type Lexer struct {
	input        string   // the string being scanned
	position     int      // current position in the input (points to current char)
	readPosition int      // current reading position in the input (after current char)
	ch           rune     // current char under examination
	pos          Position // current position in the input
	lineStart    int      // start position of the current line
}

// New creates a new lexer.
func New(input string) *Lexer {
	l := &Lexer{
		input:        input,
		pos:          Position{Line: 1, Column: 0}, // Start at line 1, column 0 (will be incremented to 1 on first readChar)
		position:     -1,                           // Start before the first character
		readPosition: 0,
		lineStart:    0,
	}
	// Read the first character
	l.readChar()
	return l
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	// Capture the starting position of the token before any character is read
	// This ensures we get the correct position for the token
	startPos := Position{Line: l.pos.Line, Column: l.pos.Column}
	// Ensure we don't have column 0
	if startPos.Column == 0 {
		startPos.Column = 1
	}

	// If we're at the end of input, return EOF
	if l.ch == 0 {
		return Token{Type: EOF, Literal: "", Pos: startPos}
	}

	// Create and return the appropriate token
	switch l.ch {
	case '=':
		tok := newToken(EQ, l.ch, startPos)
		l.readChar()
		return tok
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok := Token{Type: NEQ, Literal: literal, Pos: startPos}
			l.readChar()
			return tok
		} else {
			tok := newToken(ILLEGAL, l.ch, startPos)
			l.readChar()
			return tok
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok := Token{Type: LTE, Literal: literal, Pos: startPos}
			l.readChar()
			return tok
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok := Token{Type: NEQ, Literal: literal, Pos: startPos}
			l.readChar()
			return tok
		} else {
			tok := newToken(LT, l.ch, startPos)
			l.readChar()
			return tok
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok := Token{Type: GTE, Literal: literal, Pos: startPos}
			l.readChar()
			return tok
		} else {
			tok := newToken(GT, l.ch, startPos)
			l.readChar()
			return tok
		}
	case ';':
		tok := newToken(SEMICOLON, l.ch, startPos)
		l.readChar()
		return tok
	case '(':
		tok := newToken(LPAREN, l.ch, startPos)
		l.readChar()
		return tok
	case ')':
		tok := newToken(RPAREN, l.ch, startPos)
		l.readChar()
		return tok
	case ',':
		tok := newToken(COMMA, l.ch, startPos)
		l.readChar()
		return tok
	case '+':
		tok := newToken(PLUS, l.ch, startPos)
		l.readChar()
		return tok
	case '-':
		tok := newToken(MINUS, l.ch, startPos)
		l.readChar()
		return tok
	case '*':
		tok := newToken(ASTERISK, l.ch, startPos)
		l.readChar()
		return tok
	case '/':
		tok := newToken(SLASH, l.ch, startPos)
		l.readChar()
		return tok
	case '\'':
		// Read the raw string including quotes
		raw := l.readString()
		// The closing quote is still in the input, so we need to consume it
		l.readChar() // consume the closing quote

		// Unescape the string by replacing '' with '
		// The raw string includes the surrounding quotes, so we need to remove them first
		lit := ""
		if len(raw) >= 2 {
			// Remove the surrounding quotes
			content := raw[1 : len(raw)-1]
			// Replace '' with '
			lit = strings.ReplaceAll(content, "''", "'")
		}

		return Token{Type: STRING, Literal: lit, Pos: startPos}
	case 0:
		return Token{Type: EOF, Literal: "", Pos: startPos}
	default:
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			return Token{
				Type:    LookupIdent(lit),
				Literal: lit,
				Pos:     startPos,
			}
		} else if isDigit(l.ch) {
			lit := l.readNumber()
			return Token{
				Type:    NUMBER,
				Literal: lit,
				Pos:     startPos,
			}
		} else {
			tok := newToken(ILLEGAL, l.ch, startPos)
			l.readChar()
			return tok
		}
	}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = rune(l.input[l.readPosition])
	}

	// Update position tracking
	if l.ch == '\n' {
		l.pos.Line++
		l.pos.Column = 0
	} else {
		l.pos.Column++
	}

	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return rune(l.input[l.readPosition])
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	// Use peekChar to check the next character without consuming it
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '.' {
		l.readChar()
	}
	// The position is now one past the last character of the identifier
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position

	// Read integer part
	for isDigit(l.ch) {
		l.readChar()
	}

	// Look for a fractional part
	if l.ch == '.' && isDigit(l.peekChar()) {
		// Consume the "."
		l.readChar()

		// Consume the fractional digits
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	// Look for an exponent
	if (l.ch == 'e' || l.ch == 'E') && (isDigit(l.peekChar()) ||
		(l.peekChar() == '+' && isDigit(rune(l.input[l.readPosition+1]))) ||
		(l.peekChar() == '-' && isDigit(rune(l.input[l.readPosition+1])))) {

		// Consume 'e' or 'E'
		l.readChar()

		// Optional sign
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}

		// Must have at least one digit
		if !isDigit(l.ch) {
			// Handle error: malformed number
			return l.input[position:l.position]
		}

		// Consume exponent digits
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	// Don't consume the character after the number
	return l.input[position:l.position]
}

// readString reads a string literal from the input and returns the raw string
// including the surrounding quotes and any escaped quotes.
// The position is advanced to the closing quote, which will be consumed by NextToken.
func (l *Lexer) readString() string {
	position := l.position

	for {
		l.readChar()

		if l.ch == '\'' {
			if l.peekChar() == '\'' {
				// Found an escaped single quote ('')
				l.readChar() // consume the second quote
			} else {
				// Found the closing quote
				break
			}
		} else if l.ch == 0 {
			// Handle EOF before closing quote
			break
		}
	}

	// Return the raw string including the quotes
	return l.input[position : l.position+1]
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func newToken(tokenType TokenType, ch rune, pos Position) Token {
	// Create a copy of the position to avoid mutation issues
	posCopy := Position{
		Line:   pos.Line,
		Column: pos.Column,
	}
	return Token{
		Type:    tokenType,
		Literal: string(ch),
		Pos:     posCopy,
	}
}

// Tokenize converts the input string into a slice of tokens.
func Tokenize(input string) ([]Token, error) {
	l := New(input)
	var tokens []Token

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)

		if tok.Type == EOF {
			break
		}

		if tok.Type == ILLEGAL {
			return nil, fmt.Errorf("illegal character '%s' at line %d, column %d",
				tok.Literal, tok.Pos.Line, tok.Pos.Column)
		}
	}

	return tokens, nil
}
