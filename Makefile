MODULE_PREFIX := github.com/jcrd/petri

BUILDDIR ?= builddir

SRC := gene/genes.go cell.go ctx.go env.go rng.go stats.go vm.go \
	pb/delta.pb.go

all: $(BUILDDIR)/petri-json $(BUILDDIR)/petri-web

pb/delta.pb.go: proto/delta.proto
	protoc --go_out=. --go_opt=module=$(MODULE_PREFIX) $^

$(BUILDDIR)/petri-json: cmd/petri-json/main.go $(SRC)
	mkdir -p $(BUILDDIR)
	go build -o $@ $<

$(BUILDDIR)/petri-web: cmd/petri-web/main.go $(SRC)
	mkdir -p $(BUILDDIR)
	go build -o $@ $<

run-petri-web: $(BUILDDIR)/petri-web
	$(BUILDDIR)/petri-web -index cmd/petri-web/index.html \
		-width 64 -height 64

benchmark: env_test.go $(SRC)
	go test -bench=.

clean:
	rm -fr $(BUILDDIR)
	rm -fr pb

.PHONY: all run-petri-web benchmark clean
