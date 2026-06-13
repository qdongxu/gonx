package config

// ASTNode is the root of all config AST nodes.
type ASTNode interface {
	astNode()
}

// ConfigFile is the top-level AST node for a parsed config file.
type ConfigFile struct {
	Directives []Directive
	Blocks     []BlockNode
}

func (c *ConfigFile) astNode() {}

// Directive is a simple name-value pair terminated by semicolon.
type Directive struct {
	Name   string
	Params []string
	Line   int
}

func (d *Directive) astNode() {}

// BlockNode is a named block with an optional parameter and nested body.
type BlockNode struct {
	Name       string
	Param      string
	Directives []Directive
	Blocks     []BlockNode
	Line       int
}

func (b *BlockNode) astNode() {}

// IncludeNode represents an include directive with a glob path.
type IncludeNode struct {
	Path string
	Line int
}

func (i *IncludeNode) astNode() {}
