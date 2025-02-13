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
      php_fastcgi localhost:7071 {
        # 添加文件网络功能
        # env WASI_NET bypass=127.0.0.1
      }
    }
   ```
   运行 `caddyserver`:
   ```sh
   caddy run --watch
   ```
4. 另起终端运行 `go-wagi`:
   ```sh
   go run .
   ```
5. 大功告成, 打开下列网址测试吧:
   - [`http://127.0.0.1:7070/`](http://127.0.0.1:7070)
   - [`/hello1`](http://127.0.0.1:7070/hello1)
   - [`/hello2`](http://127.0.0.1:7070/hello2)

### WCGI 模式

当 wasm module 的 export functions 中含有 `wagi_wcgi`, 会启用该模式,
该模式复用进程, 可以将 golang wasm 的 qps 由 cgi 的 98 提高至 2380, 提高 20 倍性能

具体查看 [example.go](./example/example.go), 使用 [`wcgi`](https://github.com/shynome/wcgi) 自动适配

## Todo

- [ ] 支持资源限制
- [ ] 支持通过网络调用自身 API, 由于读取文件会导致程序挂起, 这目前不可实现, 等待 wazero 实现 [support non-blocking files](https://github.com/tetratelabs/wazero/issues/1500)
- [x] `WASI_NET` 白名单支持, 规则参考 [gost bypass](https://gost.run/concepts/bypass/)
