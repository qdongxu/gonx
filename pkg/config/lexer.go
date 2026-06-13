package config

import (
	"bufio"
	"io"
	"strings"
	"unicode"
)

// lexer scans nginx config text into tokens.
type lexer struct {
	reader *bufio.Reader
	line   int
	ch     rune
}

// newLexer creates a lexer from an io.Reader.
func newLexer(r io.Reader) *lexer {
	l := &lexer{reader: bufio.NewReader(r), line: 1}
	l.readChar()
	return l
}

// readChar advances to the next rune.
func (l *lexer) readChar() {
	ch, _, err := l.reader.ReadRune()
	if err != nil {
		l.ch = 0
		return
	}
	l.ch = ch
	if ch == '\n' {
		l.line++
	}
}

// peek returns the next rune without consuming it.
func (l *lexer) peek() rune {
	ch, _, err := l.reader.ReadRune()
	if err != nil {
		return 0
	}
	l.reader.UnreadRune()
	return ch
}

// skipWhitespace advances past spaces and tabs (not newlines).
func (l *lexer) skipWhitespace() {
	for l.ch != 0 && (l.ch == ' ' || l.ch == '\t' || l.ch == '\r') {
		l.readChar()
	}
}

// nextToken produces the next token from the input.
func (l *lexer) nextToken() Token {
	l.skipWhitespace()

	switch l.ch {
	case 0:
		return Token{Type: TokenEOF, Line: l.line}
	case ';':
		tok := Token{Type: TokenSemicolon, Value: ";", Line: l.line}
		l.readChar()
		return tok
	case '{':
		tok := Token{Type: TokenLBrace, Value: "{", Line: l.line}
		l.readChar()
		return tok
		case '}':
		tok := Token{Type: TokenRBrace, Value: "}", Line: l.line}
		l.readChar()
		return tok
	case '#':
		return l.readComment()
	case '"':
		return l.readString()
	case '\'':
		return l.readString()
	default:
		if isIdentStart(l.ch) {
			return l.readIdent()
		}
		if unicode.IsDigit(l.ch) {
			return l.readNumber()
		}
		// Skip unknown character and continue
		l.readChar()
		return l.nextToken()
	}
}

// readComment reads until end of line.
func (l *lexer) readComment() Token {
	line := l.line
	var b strings.Builder
	l.readChar() // consume '#'
	for l.ch != 0 && l.ch != '\n' {
		b.WriteRune(l.ch)
		l.readChar()
	}
	return Token{Type: TokenComment, Value: b.String(), Line: line}
}

// readString reads a quoted string.
func (l *lexer) readString() Token {
	line := l.line
	quote := l.ch
	l.readChar()
	var b strings.Builder
	for l.ch != 0 && l.ch != quote {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			default:
				b.WriteRune(l.ch)
			}
		} else {
			b.WriteRune(l.ch)
		}
		l.readChar()
	}
	if l.ch == quote {
		l.readChar()
	}
	return Token{Type: TokenString, Value: b.String(), Line: line}
}

// readIdent reads an identifier or unquoted value.
func (l *lexer) readIdent() Token {
	line := l.line
	var b strings.Builder
	for l.ch != 0 && !isDelimiter(l.ch) {
		b.WriteRune(l.ch)
		l.readChar()
	}
	val := b.String()
	return Token{Type: TokenIdent, Value: val, Line: line}
}

// readNumber reads a numeric literal.
func (l *lexer) readNumber() Token {
	line := l.line
	var b strings.Builder
	for l.ch != 0 && (unicode.IsDigit(l.ch) || l.ch == '.') {
		b.WriteRune(l.ch)
		l.readChar()
	}
	return Token{Type: TokenNumber, Value: b.String(), Line: line}
}

// isIdentStart reports whether ch can start an identifier.
func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_' || ch == '$' || ch == '~' || ch == '/' || ch == '.' || ch == '-'
}

// isDelimiter reports whether ch terminates a token.
func isDelimiter(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
		ch == ';' || ch == '{' || ch == '}' || ch == 0
}

// LexAll lexes the entire input and returns all tokens.
func LexAll(r io.Reader) ([]Token, error) {
	l := newLexer(r)
	var tokens []Token
	for {
		tok := l.nextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}
