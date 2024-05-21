build:
	CGO_ENABLED=0 go build -ldflags="-X 'main.Version=$$(git describe --tags --always --dirty)'" .
build-demo:
	GOOS=wasip1 GOARCH=wasm go build -o ./example/index.php ./example
wcgi: build-demo
	cd ./example && \
	wasm-merge -all index.php m wcgi.wat m -o index.php
caddy:
	caddy run --watch
demo: build-demo caddy
