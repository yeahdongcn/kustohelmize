VERSION    := v0.4.0
GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_SHA    := $(shell git rev-parse --short HEAD)
GIT_TAG    := $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY  := $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")