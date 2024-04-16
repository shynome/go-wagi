build:
	go build -o wagi -ldflags="-X 'main.Version=$$(git describe --tags --always --dirty)'" ./cmd/wagi
build-tinygo-demo:
	tinygo build -o ./example/index.php -target wasi ./example
build-demo:
	GOOS=wasip1 GOARCH=wasm go build -o ./example/index.php ./example
caddy:
	caddy run -watch
demo: build-demo caddy
