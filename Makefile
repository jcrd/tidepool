BUILDDIR ?= builddir

SRC := gene/genes.go cell.go ctx.go env.go rng.go stats.go vm.go

all: $(BUILDDIR)/json $(BUILDDIR)/web

$(BUILDDIR)/json: cmd/json/main.go $(SRC)
	mkdir -p $(BUILDDIR)
	go build -o $@ $<

$(BUILDDIR)/web: cmd/web/main.go $(SRC)
	mkdir -p $(BUILDDIR)
	go build -o $@ $<

run-web: $(BUILDDIR)/web
	$(BUILDDIR)/web -index cmd/web/index.html \
		-width 32 -height 32 -scale 10

benchmark: env_test.go $(SRC)
	go test -bench=.

clean:
	rm -fr $(BUILDDIR)

.PHONY: all run-web benchmark clean
