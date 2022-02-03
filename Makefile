

.PHONY: fix-alignment
fix-alignment:
	go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	fieldalignment -fix ./...
	go mod tidy