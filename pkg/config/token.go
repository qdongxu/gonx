package config

// TokenType represents the lexical token kinds.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenString
	TokenNumber
	TokenSemicolon
	TokenLBrace
	TokenRBrace
	TokenComment
)

// Token is a lexical unit produced by the lexer.
type Token struct {
	Type  TokenType
	Value string
	Line  int
}

// String returns a human-readable token name.
func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "IDENT"
	case TokenString:
		return "STRING"
	case TokenNumber:
		return "NUMBER"
	case TokenSemicolon:
		return "SEMICOLON"
	case TokenLBrace:
		return "LBRACE"
	case TokenRBrace:
		return "RBRACE"
	case TokenComment:
		return "COMMENT"
	default:
		return "UNKNOWN"
	}
}
