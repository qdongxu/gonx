package config

import (
	"fmt"
	"io"
	"strings"
)

// parser implements a recursive-descent parser for nginx config syntax.
type parser struct {
	lexer  *lexer
	cur    Token
	errors []string
}

// NginxParser implements ConfigParser for nginx configuration files.
type NginxParser struct{}

// Parse reads nginx config syntax and produces a Config AST.
func (n *NginxParser) Parse(r io.Reader) (*Config, error) {
	p := newParser(r)
	ast, err := p.parse()
	if err != nil {
		return nil, err
	}
	return astToConfig(ast), nil
}

// NewNginxParser creates a ConfigParser that parses nginx syntax.
func NewNginxParser() ConfigParser {
	return &NginxParser{}
}

// newParser creates a parser from an io.Reader.
func newParser(r io.Reader) *parser {
	l := newLexer(r)
	p := &parser{lexer: l}
	p.advance()
	return p
}

// advance moves to the next non-comment token.
func (p *parser) advance() {
	for {
		p.cur = p.lexer.nextToken()
		if p.cur.Type != TokenComment {
			break
		}
	}
}

// expect checks that the current token is of the given type and advances.
func (p *parser) expect(tt TokenType) error {
	if p.cur.Type != tt {
		p.errors = append(p.errors,
			fmt.Sprintf("line %d: expected %s, got %s (%s)",
				p.cur.Line, tt, p.cur.Type, p.cur.Value))
		return fmt.Errorf("parse error at line %d", p.cur.Line)
	}
	p.advance()
	return nil
}

// parse parses the entire config file.
func (p *parser) parse() (*ConfigFile, error) {
	cfg := &ConfigFile{}
	for p.cur.Type != TokenEOF {
		if err := p.parseTopLevel(cfg); err != nil {
			return nil, err
		}
	}
	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse errors: %s", strings.Join(p.errors, "; "))
	}
	return cfg, nil
}

// parseTopLevel parses a top-level directive or block.
func (p *parser) parseTopLevel(cfg *ConfigFile) error {
	if p.cur.Type != TokenIdent {
		return fmt.Errorf("line %d: unexpected token %s", p.cur.Line, p.cur.Value)
	}
	name := p.cur.Value
	line := p.cur.Line
	p.advance()

	params := p.parseParams()

	if p.cur.Type == TokenLBrace {
		block, err := p.parseBlock(name, params, line)
		if err != nil {
			return err
		}
		cfg.Blocks = append(cfg.Blocks, block)
		return nil
	}

	if err := p.expect(TokenSemicolon); err != nil {
		return err
	}
	cfg.Directives = append(cfg.Directives, Directive{
		Name:   name,
		Params: params,
		Line:   line,
	})
	return nil
}

// parseParams reads zero or more parameter tokens until a delimiter.
func (p *parser) parseParams() []string {
	var params []string
	for p.cur.Type == TokenIdent || p.cur.Type == TokenString || p.cur.Type == TokenNumber {
		params = append(params, p.cur.Value)
		p.advance()
	}
	return params
}

// parseBlock parses a block body after the opening brace.
func (p *parser) parseBlock(name string, params []string, line int) (BlockNode, error) {
	if err := p.expect(TokenLBrace); err != nil {
		return BlockNode{}, err
	}
	block := BlockNode{Name: name, Line: line}
	if len(params) > 0 {
		block.Param = params[0]
	}
	for p.cur.Type != TokenRBrace && p.cur.Type != TokenEOF {
		if err := p.parseBlockBody(&block); err != nil {
			return BlockNode{}, err
		}
	}
	if err := p.expect(TokenRBrace); err != nil {
		return BlockNode{}, err
	}
	return block, nil
}

// parseBlockBody parses a directive or nested block inside a block.
func (p *parser) parseBlockBody(parent *BlockNode) error {
	if p.cur.Type != TokenIdent {
		return fmt.Errorf("line %d: unexpected token %s", p.cur.Line, p.cur.Value)
	}
	name := p.cur.Value
	line := p.cur.Line
	p.advance()

	params := p.parseParams()

	if p.cur.Type == TokenLBrace {
		block, err := p.parseBlock(name, params, line)
		if err != nil {
			return err
		}
		parent.Blocks = append(parent.Blocks, block)
		return nil
	}

	if err := p.expect(TokenSemicolon); err != nil {
		return err
	}
	parent.Directives = append(parent.Directives, Directive{
		Name:   name,
		Params: params,
		Line:   line,
	})
	return nil
}

// astToConfig converts the AST to the legacy Config structure.
func astToConfig(ast *ConfigFile) *Config {
	cfg := &Config{Version: "0.0.0", Blocks: []Block{}}
	for _, d := range ast.Directives {
		cfg.Blocks = append(cfg.Blocks, Block{
			Type:   d.Name,
			Params: mapFromParams(d.Params),
		})
	}
	for _, b := range ast.Blocks {
		cfg.Blocks = append(cfg.Blocks, astBlockToBlock(b))
	}
	return cfg
}

// astBlockToBlock recursively converts a BlockNode to the legacy Block.
func astBlockToBlock(b BlockNode) Block {
	params := map[string]string{"_type": b.Name}
	if b.Param != "" {
		params["_param"] = b.Param
	}
	for _, d := range b.Directives {
		if len(d.Params) > 0 {
			params[d.Name] = d.Params[0]
		} else {
			params[d.Name] = ""
		}
	}
	for _, nb := range b.Blocks {
		nested := astBlockToBlock(nb)
		for k, v := range nested.Params {
			if k == "_type" || k == "_param" {
				continue
			}
			params[k] = v
		}
	}
	return Block{Type: b.Name, Params: params}
}

// mapFromParams converts a string slice to a map (first param as key if odd).
func mapFromParams(params []string) map[string]string {
	m := make(map[string]string)
	for i := 0; i < len(params); i += 2 {
		if i+1 < len(params) {
			m[params[i]] = params[i+1]
		} else {
			m[params[i]] = ""
		}
	}
	return m
}
