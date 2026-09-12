.PHONY: deps dev build test race bench check install uninstall autostart autostart-off smoke preview

deps:
	go mod download
	cd frontend && npm ci --no-audit --no-fund

dev:
	wails dev -tags webkit2_41

build:
	wails build -tags webkit2_41

test:
	go test ./internal/...

race:
	go test -race ./internal/...

bench:
	go test -run '^$$' -bench=Search -benchmem ./internal/catalog

check: test
	cd frontend && npm run build
	go vet -tags webkit2_41 ./...

install: build
	python3 scripts/install.py install

uninstall:
	python3 scripts/install.py uninstall

autostart:
	python3 scripts/install.py autostart

autostart-off:
	python3 scripts/install.py autostart-off

smoke: build
	python3 scripts/native-smoke.py

preview:
	cd frontend && npm run dev -- --host 127.0.0.1
