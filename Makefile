# DEPRECATED: This Makefile has been replaced by Taskfile.yml
# 
# Please install Task: https://taskfile.dev/installation/
# Then use `task` instead of `make`
#
# Quick command mapping:
# - make          → task (or task test)
# - make test     → task test
# - make testint  → task testint
# - make fmt      → task fmt
# - make fmtcheck → task fmtcheck
# - make lint     → task lint
# - make deeplint → task deeplint
# - make deps-start → task deps-start
# - make deps-stop  → task deps-stop
# - make example-service → task example-service
# - make example-client  → task example-client
# - make ci       → task ci
#
# List all available tasks: task --list

.PHONY: help
help:
	@echo "DEPRECATED: This Makefile has been replaced by Taskfile.yml"
	@echo ""
	@echo "Please install Task: https://taskfile.dev/installation/"
	@echo "Then use 'task' instead of 'make'"
	@echo ""
	@echo "List all available tasks: task --list"

.DEFAULT_GOAL := help
