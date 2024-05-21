/*
Copyright © 2024 shynome <shynome@gmail.com>
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"github.com/shynome/go-wagi/cgi"
	"github.com/shynome/go-wagi/fsnet"
	"github.com/shynome/wcgi"
	"github.com/spf13/cobra"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var args struct {
	listen string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-wagi",
	Short: "wasm cgi 的 fastcgi server",
	Long:  `wasm cgi 的 fastcgi server`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, _args []string) {

		l := try.To1(net.Listen("tcp", args.listen))
		defer l.Close()

		ctx := context.Background()
		waCache := try.To1(wazero.NewCompilationCacheWithDir(".wazero"))
		rtc := wazero.NewRuntimeConfig().
			WithCompilationCache(waCache).
			WithCloseOnContextDone(true)
		rt := wazero.NewRuntimeWithConfig(ctx, rtc)
		wasi_snapshot_preview1.MustInstantiate(ctx, rt)

		mCache := newCache[func() (*WasmItem, error)]()
		proxyCache := newCache[func() (*ProxyItem, error)]()
		instCache := newCache[*InstanceItem]()

		var keepAlive = 10 * time.Minute
		srv := http.NewServeMux()
		srv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			var err error
			defer err0.Then(&err, nil, func() {
				log.Println("err", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			})

			env := fcgi.ProcessEnv(r)
			script := env["SCRIPT_FILENAME"]
			cwd := env["DOCUMENT_ROOT"]
			finfo := try.To1(os.Stat(script))

			fileKey := "file-" + script
			wasmKey := fmt.Sprintf("file-%s-%d", script, finfo.ModTime().Unix())
			netRule := env["WASI_NET"]
			proxyKey := strings.Join([]string{wasmKey, env["WASI_DEBUG"], cwd, netRule}, ",")

			inst := instCache.Get(fileKey)
			if inst == nil {
				func() {
					instCache.mux.Lock()
					defer instCache.mux.Unlock()
					ctx := context.Background()
					ctx, cancel := context.WithCancel(ctx)
					timer := time.AfterFunc(keepAlive, func() {
						cancel()
					})
					go func() {
						<-ctx.Done()
						instCache.Del(fileKey)
					}()
					inst = &InstanceItem{
						WasmKey:  wasmKey,
						ProxyKey: proxyKey,
						timer:    timer,
						ctx:      ctx,
					}
				}()
				instCache.Set(fileKey, inst)
			} else {
				inst.timer.Reset(keepAlive)
			}

			wasmGet := mCache.Get(wasmKey)
			proxyGet := proxyCache.Get(proxyKey)
			func() {
				instCache.mux.RLock()
				defer instCache.mux.RUnlock()

				// clear old wasm module
				func() {
					if inst.WasmKey == wasmKey {
						return
					}
					wasmGet := mCache.Get(inst.WasmKey)
					if wasmGet == nil {
						return
					}
					if mod, err := wasmGet(); err == nil {
						mod.Close(context.Background())
					}
					mCache.Del(wasmKey)
				}()
				// clear old proxy instance
				func() {
					if inst.ProxyKey == proxyKey {
						return
					}
					proxyGet := proxyCache.Get(inst.ProxyKey)
					if proxyGet == nil {
						return
					}
					if proxy, err := proxyGet(); err == nil {
						proxy.Close()
					}
					mCache.Del(proxyKey)
				}()
				inst.WasmKey = wasmKey
				inst.ProxyKey = proxyKey
			}()

			if wasmGet == nil {
				wasmGet = sync.OnceValues(func() (*WasmItem, error) {
					binary, err := os.ReadFile(script)
					if err != nil {
						return nil, err
					}
					ctx := inst.ctx
					ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
					defer cancel()
					mod, err := rt.CompileModule(ctx, binary)
					if err != nil {
						return nil, err
					}
					_, wcgi := mod.ExportedFunctions()["wagi_wcgi"]
					return &WasmItem{
						CompiledModule: mod,
						SupportWCGI:    wcgi,
					}, nil
				})
				mCache.Set(wasmKey, wasmGet)
			}
			wasm, err := wasmGet()
			if err != nil {
				mCache.Del(wasmKey)
				return
			}

			// 强制以 CGI 模式运行
			forceCGI := env["WASI_CGI"] == "true"
			if forceCGI || !wasm.SupportWCGI {
				envList := []string{}
				for k, v := range env {
					envList = append(envList, k+"="+v)
				}

				h := cgi.Handler{
					Path:   script,
					Args:   []string{"wcgi"},
					Env:    envList,
					Dir:    cwd,
					Stderr: os.Stderr,

					Runtime: rt,
					WASM:    wasm.CompiledModule,
				}
				h.ServeHTTP(w, r)
				return
			}

			if proxyGet == nil {
				proxyGet = sync.OnceValues(func() (_ *ProxyItem, err error) {
					ctx := inst.ctx
					ctx, cancel := context.WithCancel(ctx)
					go func() {
						<-ctx.Done()
						proxyCache.Del(proxyKey)
					}()

					defer err0.Then(&err, nil, func() {
						cancel()
					})

					stdio := &wcgi.Stdio{}
					var (
						stdin  io.Reader
						stdout io.Writer
					)
					stdin, stdio.Writer = try.To2(os.Pipe())
					stdio.Reader, stdout = try.To2(os.Pipe())

					mc := wazero.NewModuleConfig()
					mc = cgi.WithCommonConfig(mc)
					fsc := wazero.NewFSConfig()
					if cwd != "" {
						fsc = fsc.WithDirMount(cwd, cwd)
					}
					if netRule != "" {
						fsc = fsc.WithFSMount(fsnet.New(netRule), "/dev")
					}
					mc = mc.WithFSConfig(fsc)
					env["WAGI_WCGI"] = "true"
					for k, v := range env {
						mc = mc.WithEnv(k, v)
					}
					if env["WASI_DEBUG"] == "true" {
						mc = mc.WithStderr(os.Stderr)
					}
					mc = mc.WithStdin(stdin).WithStdout(stdout)

					go func() {
						defer cancel()
						mc := mc.WithName("")
						mod, err := rt.InstantiateModule(ctx, wasm.CompiledModule, mc)
						if err != nil {
							return
						}
						defer mod.Close(ctx)
					}()

					yc := yamux.DefaultConfig()
					yc.KeepAliveInterval = 10 * time.Second
					yc.StreamCloseTimeout = 5 * time.Second
					yc.StreamOpenTimeout = 5 * time.Second
					sess := try.To1(yamux.Client(stdio, yc))
					sess.Ping()
					go func() {
						<-sess.CloseChan()
						cancel()
					}()
					go func() {
						<-ctx.Done()
						sess.Close()
					}()

					endpoint := fmt.Sprintf("http://yamux.proxy/")
					target := try.To1(url.Parse(endpoint))
					handler := httputil.NewSingleHostReverseProxy(target)
					handler.Transport = &http.Transport{
						DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
							conn, err := sess.Open()
							return conn, err
						},
					}
					go http.Serve(sess, handler)

					proxy := &ProxyItem{
						Handler: handler,
						Close:   cancel,
					}
					return proxy, nil
				})
				proxyCache.Set(proxyKey, proxyGet)
			}

			proxy, err := proxyGet()
			if err != nil {
				proxyCache.Del(proxyKey)
				return
			}

			proxy.ServeHTTP(w, r)
		})

		slog.Warn("server is running", "addr", l.Addr())
		try.To(fcgi.Serve(l, srv))
	},
}

func getWASMTry(ctx context.Context, rt wazero.Runtime, script string) wazero.CompiledModule {
	wasm := try.To1(os.ReadFile(script))
	m := try.To1(rt.CompileModule(ctx, wasm))
	return m
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
	rootCmd.Version = version
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.go-wagi.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringVar(&args.listen, "listen", "127.0.0.1:7071", "listen addr")
}

type InstanceItem struct {
	WasmKey  string
	ProxyKey string
	ctx      context.Context
	timer    *time.Timer
}

type WasmItem struct {
	wazero.CompiledModule
	SupportWCGI bool
}

type ProxyItem struct {
	http.Handler
	Close func()
}
