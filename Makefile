build-demo:
	tinygo build -o ./example/index.php -scheduler=none -target=wasi ./example
caddy:
	caddy run -watch
demo: build-demo caddy
