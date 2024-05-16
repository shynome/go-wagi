build:
	go build -o wagi -ldflags="-X 'main.Version=$$(git describe --tags --always --dirty)'" ./cmd/wagi
build-demo:
	GOOS=wasip1 GOARCH=wasm go build -o ./example/index.php ./example
wcgi: build-demo
	cd ./example && \
	wasm-merge -all index.php m wcgi.wat m -o index.php
caddy:
	caddy run --watch
demo: build-demo caddy
