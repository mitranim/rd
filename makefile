MAKEFLAGS := --silent --always-make
MAKE_PAR := $(MAKE) -j 128
GO_FLAGS := -tags=$(tags)
TEST_FLAGS := $(if $(filter $(verb), true), -v,) -count=1 $(GO_FLAGS)
TEST := test $(TEST_FLAGS) -timeout=1s -run=$(run)
BENCH := test $(TEST_FLAGS) -run=- -bench=$(or $(run),.) -benchmem
WATCH := watchexec -r -c -d=0 -n

test_w:
	gow -c -v $(TEST)

test:
	go $(TEST)

bench_w:
	gow -c -v $(BENCH)

bench:
	go $(BENCH)

lint_w:
	$(WATCH) -- $(MAKE) lint

lint:
	golangci-lint run
	echo [lint] ok

prep:
	$(MAKE_PAR) test lint

release: prep
ifeq ($(tag),)
	$(error missing tag)
endif
	git pull --ff-only
	git show-ref --tags --quiet "$(tag)" || git tag "$(tag)"
	git push origin $$(git symbolic-ref --short HEAD) "$(tag)"
