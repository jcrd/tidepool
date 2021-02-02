BUILDDIR ?= builddir

SRC := gene/genes.go cell.go ctx.go env.go rng.go stats.go vm.go

all: $(BUILDDIR)/petri-json $(BUILDDIR)/petri-web

$(BUILDDIR)/petri-json: cmd/petri-json/main.go $(SRC)
	mkdir -p $(BUILDDIR)
	go build -o $@ $<

$(BUILDDIR)/petri-web: cmd/petri-web/main.go $(SRC)
	mkdir -p $(BUILDDIR)
	go build -o $@ $<

run-petri-web: $(BUILDDIR)/petri-web
	$(BUILDDIR)/petri-web -index cmd/petri-web/index.html \
		-width 64 -height 64

clean:
	rm -fr $(BUILDDIR)

.PHONY: all run-petri-web clean
