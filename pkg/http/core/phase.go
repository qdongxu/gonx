package core

// Phase represents a stage in nginx-style request processing.
type Phase int

const (
	PostReadPhase Phase = iota
	ServerRewritePhase
	FindConfigPhase
	RewritePhase
	PostRewritePhase
	PreaccessPhase
	AccessPhase
	PostAccessPhase
	TryFilesPhase
	ContentPhase
	LogPhase
)

var phaseNames = []string{
	"post_read", "server_rewrite", "find_config", "rewrite",
	"post_rewrite", "preaccess", "access", "post_access",
	"try_files", "content", "log",
}

// String returns the human-readable name of a Phase.
func (p Phase) String() string {
	if int(p) < len(phaseNames) {
		return phaseNames[p]
	}
	return "unknown"
}

// PhaseHandler runs at a specific phase.
type PhaseHandler interface {
	Phase() Phase
	Handle(req *Request) *Response
}

// PhaseEngine drives the request through the phase pipeline.
type PhaseEngine interface {
	Register(h PhaseHandler) error
	Run(ctx *PhaseContext) *Response
}

// PhaseContext holds state for a single request.
type PhaseContext struct {
	Conn     *Conn
	Server   *Server
	Handler  Handler
	Request  *Request
	Response *Response
	Vars     map[string]string
}

// Var returns a phase-local variable.
func (ctx *PhaseContext) Var(name string) string {
	if ctx.Vars == nil {
		return ""
	}
	return ctx.Vars[name]
}

// SetVar sets a phase-local variable.
func (ctx *PhaseContext) SetVar(name, value string) {
	if ctx.Vars == nil {
		ctx.Vars = make(map[string]string)
	}
	ctx.Vars[name] = value
}

// DefaultPhaseEngine is a simple PhaseEngine implementation.
type DefaultPhaseEngine struct {
	handlers map[Phase][]PhaseHandler
}

// NewDefaultPhaseEngine creates a new DefaultPhaseEngine.
func NewDefaultPhaseEngine() *DefaultPhaseEngine {
	return &DefaultPhaseEngine{handlers: make(map[Phase][]PhaseHandler)}
}

// Register adds a PhaseHandler to the engine.
func (e *DefaultPhaseEngine) Register(h PhaseHandler) error {
	p := h.Phase()
	e.handlers[p] = append(e.handlers[p], h)
	return nil
}

// Run executes the phase pipeline.
func (e *DefaultPhaseEngine) Run(ctx *PhaseContext) *Response {
	if ctx.Handler != nil && ctx.Request != nil {
		return ctx.Handler.Handle(ctx.Request)
	}
	return nil
}
