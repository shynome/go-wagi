package wagi

import (
	"context"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

type WASIRuntime interface {
	Load(path string) (module wazero.CompiledModule, err error)
	Run(ctx context.Context, path string, config wazero.ModuleConfig) (err error)
	Unload(path string) (err error)
}

var _ WASIRuntime = (*WAZeroRuntime)(nil)

type WAZeroRuntime struct {
	l       *sync.RWMutex
	runtime wazero.Runtime
	codes   *ttlcache.Cache[string, *Item]
}

type WagiConfig struct {
	CacheCapacity uint64
	CacheTTL      time.Duration
}

func NewWagi(cfg WagiConfig) *WAZeroRuntime {
	ctx := context.Background()
	rtc := wazero.NewRuntimeConfigInterpreter().
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(16 * 20) // 10M meomory limit
	runtime := wazero.NewRuntimeWithConfig(ctx, rtc)
	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	var codes *ttlcache.Cache[string, *Item]
	{ // codes cache
		var items = Items{Runtime: runtime}
		var cacheOpts = []ttlcache.Option[string, *Item]{
			ttlcache.WithLoader[string, *Item](items),
		}

		if cfg.CacheCapacity > 0 {
			cacheOpts = append(cacheOpts, ttlcache.WithCapacity[string, *Item](cfg.CacheCapacity))
		}

		if cfg.CacheTTL == 0 {
			cfg.CacheTTL = 15 * time.Second
		}
		if cfg.CacheTTL > 0 {
			cacheOpts = append(cacheOpts, ttlcache.WithTTL[string, *Item](cfg.CacheTTL))
		}

		codes = ttlcache.New(cacheOpts...)
		go codes.Start()
	}

	wr := &WAZeroRuntime{
		runtime: runtime,
		codes:   codes,
	}

	return wr
}

func (w *WAZeroRuntime) Load(path string) (module wazero.CompiledModule, err error) {
	defer func() {
		if err != nil {
			w.codes.Delete(path)
		}
	}()
	item := w.codes.Get(path)
	v := item.Value()
	v.locker.RLock()
	defer v.locker.RUnlock()
	if module, err = v.compiled, v.err; err != nil {
		return
	}
	if err = v.Expired(); err != nil {
		return
	}
	w.codes.Set(path, v, 0)
	return
}

func (w *WAZeroRuntime) Run(ctx context.Context, path string, config wazero.ModuleConfig) (err error) {
	defer err2.Handle(&err, func() {
		w.codes.Delete(path)
	})

	item := w.codes.Get(path).Value()
	try.To(item.Error())

	config = config.
		WithName("")
	var m api.Module
	m, err = w.runtime.InstantiateModule(ctx, item.compiled, config)
	m.Close(ctx)

	if e, ok := err.(*sys.ExitError); ok {
		if code := e.ExitCode(); code == 0 {
			return nil
		}
	}
	return
}

func (w *WAZeroRuntime) Unload(path string) (err error) {
	w.codes.Delete(path)
	return
}
