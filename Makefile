# Os dois motores (PROJECT.md, "Dois motores"): o go e o Galaxy tem de dar o mesmo resultado.
GALAXY ?= galaxy
GALAXY_MEM ?= 6G
PKGS ?= ./...

.PHONY: test test-go test-galaxy test-ui test-desktop test-p2p p2p-lib vet ui desktop

test: vet test-go test-galaxy test-ui test-desktop test-p2p

# A janela nativa (desktop/, Rust): a libghostty-vt compila a Ghostty com o Zig 0.15 (ZIG), e o
# target fica fora da arvore (CARGO_TARGET_DIR), que e' grande.
ZIG ?= $(HOME)/.local/share/bavel-oraculos/zig-x86_64-linux-0.15.2
CARGO_TARGET_DIR ?= $(HOME)/.cache/polvo-desktop-target
CARGO = PATH="$(ZIG):$(HOME)/.cargo/bin:$$PATH" CARGO_TARGET_DIR="$(CARGO_TARGET_DIR)" cargo

desktop:
	cd desktop && $(CARGO) build --release

test-desktop:
	cd desktop && $(CARGO) test

# O transporte entre redes (p2p/, Rust, sobre o iroh), com o target dele fora da arvore.
P2P_TARGET_DIR ?= $(HOME)/.cache/polvo-p2p-target
P2P_CARGO = PATH="$(HOME)/.cargo/bin:$$PATH" CARGO_TARGET_DIR="$(P2P_TARGET_DIR)" cargo

# A libpolvo_p2p.a (a API C) e os testes do lado Go com o transporte (a etiqueta polvo_p2p e o cgo).
p2p-lib:
	cd p2p && $(P2P_CARGO) build --release

test-p2p: p2p-lib
	cd p2p && $(P2P_CARGO) test
	CGO_LDFLAGS="-L$(P2P_TARGET_DIR)/release" go test -tags polvo_p2p ./internal/transport/... ./internal/station/... ./internal/node/...

.PHONY: report-p2p
report-p2p:
	@mkdir -p .anchors
	cd p2p && $(P2P_CARGO) nextest run --profile default --no-fail-fast; \
	  cp target/nextest/default/junit.xml ../.anchors/junit-p2p.xml

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

# O JUnit da janela nativa (o nextest escreve-o em desktop/target/nextest, fora do git).
.PHONY: report-desktop
report-desktop:
	@mkdir -p .anchors
	cd desktop && $(CARGO) nextest run --profile default --no-fail-fast; \
	  cp target/nextest/default/junit.xml ../.anchors/junit-desktop.xml
