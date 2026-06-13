# Gonx

A Go reimplementation of Nginx — migration plan in progress.

## Build

Requires Go 1.22 or later.

```bash
make build    # build binary
make test     # run tests
make lint     # fmt + vet
make clean    # remove binary
```

## Run

```bash
./gonx -c /path/to/config.conf
```

## Configuration Parser

Phase 0 introduces a skeleton nginx-compatible configuration parser.

```go
import "github.com/qdongxu/gonx/pkg/config"

parser := config.NewParser()
cfg, err := parser.Parse(strings.NewReader(`
    worker_processes 1;
    events {
        worker_connections 1024;
    }
    http {
        server {
            listen 80;
            location / {
                root /var/www;
            }
        }
    }
`))
```

Supported syntax:
- Directives: `name value1 value2;`
- Blocks: `name { ... }`
- Quoted strings: `"path/with spaces"`
- Comments: `# comment until end of line`
- Nested blocks (server, location, upstream, etc.)

## Status

Phase 0: project skeleton and nginx config parser skeleton.

## License

MIT (see LICENSE)
