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

## Status

Phase 0: project skeleton and module interfaces.

## License

MIT (see LICENSE)
