# Os dois motores (PROJECT.md, "Dois motores"): o go e o Galaxy tem de dar o mesmo resultado.
GALAXY ?= galaxy
GALAXY_MEM ?= 6G
PKGS ?= ./...

.PHONY: test test-go test-galaxy test-ui vet ui

test: vet test-go test-galaxy test-ui

# A UI (ui/): o bundle vai para internal/web/dist, embutido no executavel.
ui:
	cd ui && npm ci && npm run build

test-ui:
	cd ui && npm test && npm run typecheck

vet:
	go vet $(PKGS)

test-go:
	go test -count=1 $(PKGS)

# Sob um teto de memoria (o systemd-run), para um defeito do Galaxy nao levar a maquina.
test-galaxy:
	@for p in $$(go list -f "{{if .TestGoFiles}}{{.Dir}}{{end}}" $(PKGS)); do \
	  echo "galaxy test $$p"; \
	  systemd-run --user --scope -p MemoryMax=$(GALAXY_MEM) -p MemorySwapMax=0 --quiet \
	    env GALAXY_ENGINE=go $(GALAXY) test $$p || exit 1; \
	done

# O relatorio que o `anchors test` ingere (anchors.yaml, `tests:`): o JUnit prova os cenarios,
# o lcov mede as linhas. Corre no go; o galaxy e' o `test-galaxy`.
.PHONY: report
report:
	@mkdir -p .anchors
	go test -count=1 -v -coverpkg=./internal/...,./cmd/... -coverprofile=.anchors/cover.out ./... 2>&1 \
	  | go tool go-junit-report -set-exit-code > .anchors/junit.xml
	go tool gcov2lcov -infile .anchors/cover.out -outfile .anchors/lcov.info
