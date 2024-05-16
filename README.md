## 简述

使用 cgi 调用 wasm 不就是天然的无服务器嘛, 资源占用小, 隔离性好, 这么多优势岂能不试

## 使用

现在让我们看看, 该设想下的无服务器架构.

1. 编写服务

   ```go
    package main
    import (
      "fmt"
      "net/http"
      "net/http/cgi"
      "os"
    )

    func main() {
      http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "index")
      })
      http.HandleFunc("/hello1", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "hello1")
      })
      http.HandleFunc("/hello2", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "hello2")
      })
      // http.ListenAndServe 替换为 cgi.Serve 即可
      if err := cgi.Serve(nil); err != nil {
        fmt.Fprintln(os.Stderr, err)
      }
    }
   ```

2. 编译成 `wasm` 但输出为 `index.php`, 这样方便重用 php 的配置
   ```sh
   GOOS=wasip1 GOARCH=wasm go build -o ./example/index.php ./example
   ```
3. 添加域名访问入口
   ```Caddyfile
    # Caddyfile
    http://127.0.0.1:7070 {
      root ./example
      php_fastcgi localhost:7071
    }
   ```
   运行 `caddyserver`:
   ```sh
   caddy run -watch
   ```
4. 另起终端运行 `wasm cgi server`:
   ```sh
   go run ./cmd/wagi
   ```
5. 大功告成, 打开下列网址测试吧:
   - [`http://127.0.0.1:7070/`](http://127.0.0.1:7070)
   - [`/hello1`](http://127.0.0.1:7070/hello1)
   - [`/hello2`](http://127.0.0.1:7070/hello2)

## Todo

- [x] `WASI_NET` 白名单支持, 规则参考 [gost bypass](https://gost.run/concepts/bypass/)
