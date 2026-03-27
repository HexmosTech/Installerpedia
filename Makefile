# ----------------------------------------
# Project config
# ----------------------------------------
APP_NAME := ipm
GO       := go
SHELL    := /bin/bash

# Ensure scripts are executable
SCRIPTS := \
	build.sh \
	build_compressed.sh \
	build_and_install.sh \
	version_bump.sh \
	create_release.sh

# ----------------------------------------
# Default target
# ----------------------------------------
.DEFAULT_GOAL := help

# ----------------------------------------
# Helpers
# ----------------------------------------
.PHONY: help
help:
	@echo ""
	@echo "Available make commands:"
	@echo ""
	@echo "Build:"
	@echo "  make build                    Build ipm binary"
	@echo "  make build-compressed          Build compressed binaries"
	@echo ""
	@echo "Install:"
	@echo "  make install                  Build & install"
	@echo "  make install-compressed       Build & install (compressed)"
	@echo ""
	@echo "Release:"
	@echo "  make bump                     Bump version"
	@echo "  make release                  Create GitHub release draft"
	@echo ""
	@echo ""

# ----------------------------------------
# Script permissions
# ----------------------------------------
.PHONY: ensure-scripts
ensure-scripts:
	chmod +x $(SCRIPTS)

# ----------------------------------------
# Script-based targets
# ----------------------------------------
.PHONY: build
build: ensure-scripts
	./build.sh

.PHONY: build-compressed
build-compressed: ensure-scripts
	./build_compressed.sh

.PHONY: install
install: ensure-scripts
	./build_and_install.sh

.PHONY: bump
bump: ensure-scripts
	@# check that a sub-target is provided
ifeq ($(MAKECMDGOALS),bump)
	$(error Usage: make bump [major|minor|patch])
endif
	./version_bump.sh $(word 2,$(MAKECMDGOALS))


.PHONY: release
release: ensure-scripts
	./create_release.sh

