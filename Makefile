BUILDDIR ?= builddir

SRC := gene/genes.go cell.go ctx.go env.go rng.go vm.go

all: $(BUILDDIR)/petri-json $(BUILDDIR)/petri-web

$(BUILDDIR)/petri-json: cmd/petri-json/main.go $(SRC)
	mkdir -p $(BUILDDIR)
	go build -o $@ $<

$(BUILDDIR)/petri-web: cmd/petri-web/main.go $(SRC)
	mkdir -p $(BUILDDIR)
	go build -o $@ $<

clean:
	rm -fr $(BUILDDIR)

.PHONY: all clean
