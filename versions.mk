CLI_VERSION := 1.0.0
GIT_COMMIT  := $(shell git rev-parse HEAD)
GIT_DIRTY   := $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")