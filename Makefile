build-tinygo-demo:
	tinygo build -o ./example/index.php -target wasi ./example
build-demo:
	GOOS=js GOARCH=wasm go build -o ./example/index.php ./example
caddy:
	caddy run -watch
demo: build-demo caddy
