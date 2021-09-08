BUILDDIR ?= builddir

LIB := tidepool
SRC := $(LIB)/gene/genes.go \
	$(LIB)/cell.go \
	$(LIB)/ctx.go \
	$(LIB)/env.go \
	$(LIB)/rng.go \
	$(LIB)/stats.go \
	$(LIB)/vm.go

all: $(BUILDDIR)/json $(BUILDDIR)/web

$(BUILDDIR)/json: cmd/json/main.go $(SRC)
	mkdir -p $(BUILDDIR)
ifdef DEBUG
	go build -race -o $@ $<
else
	go build -o $@ $<
endif

$(BUILDDIR)/web: cmd/web/main.go $(SRC)
	mkdir -p $(BUILDDIR)
ifdef DEBUG
	go build -race -o $@ $<
else
	go build -o $@ $<
endif

run-web: $(BUILDDIR)/web
	$(BUILDDIR)/web \
		-index cmd/web/index.html \
		-static cmd/web/static \
		-width 32 -height 32 -scale 10

benchmark: $(LIB)/env_test.go $(SRC)
	go test ./$(LIB) -bench=.

clean:
	rm -fr $(BUILDDIR)

.PHONY: all run-web benchmark clean
