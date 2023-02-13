PATH := "${CURDIR}/bin:$(PATH)"

.PHONY: gobuildcache

bin/gotmpl: gobuildcache
	go build -o $@ ./cmd/gotmpl

bin/gotmpl-exec: gobuildcache
	go build -o $@ ./cmd/gotmpl-exec

bin/exampleplugin: gobuildcache
	go build -o $@ ./cmd/exampleplugin

bin/gotmpl-sprig: gobuildcache
	go build -o $@ ./plugins/gotmpl-sprig

bin/golangci-lint: .bindown.yml
	script/bindown install $(notdir $@)

bin/shellcheck: .bindown.yml
	script/bindown install $(notdir $@)

bin/gofumpt: .bindown.yml
	script/bindown install $(notdir $@)

bin/buf: .bindown.yml
	script/bindown install $(notdir $@)

bin/protoc: .bindown.yml
	script/bindown install $(notdir $@)

bin/protoc-gen-go: .bindown.yml
	script/bindown install $(notdir $@)

HANDCRAFTED_REV := 082e94edadf89c33db0afb48889c8419a2cb46a9
bin/handcrafted: Makefile
	GOBIN=${CURDIR}/bin \
	go install github.com/willabides/handcrafted@$(HANDCRAFTED_REV)

PROTOC_GEN_GO_GRPC_REV := v1.2.0
bin/protoc-gen-go-grpc: Makefile
	GOBIN=${CURDIR}/bin \
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_REV)

PROTOC_GEN_CONNECT_GO_REV := v1.5.1
bin/protoc-gen-connect-go: Makefile
	GOBIN=${CURDIR}/bin \
	go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@$(PROTOC_GEN_CONNECT_GO_REV)

GOIMPORTS_REV := v0.5.0
bin/goimports: Makefile
	GOBIN=${CURDIR}/bin \
	go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_REV)
