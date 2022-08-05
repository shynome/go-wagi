package wagi

import (
	"context"
	"os"
	"sync"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/rs/xid"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/wasi_snapshot_preview1"
)

type WASIRuntime interface {
	Load(path string) (module wazero.CompiledModule, err error)
	Run(path string, config wazero.ModuleConfig) (err error)
	Unload(path string) (err error)
}

var _ WASIRuntime = &WAZeroRuntime{}

type WAZeroRuntime struct {
	l       *sync.RWMutex
	runtime wazero.Runtime
	codes   map[string]wazero.CompiledModule
}

func NewWagi() *WAZeroRuntime {
	ctx := context.Background()
	runtime := wazero.NewRuntimeWithConfig(wazero.NewRuntimeConfig().WithWasmCore2())
	try.To1(wasi_snapshot_preview1.Instantiate(ctx, runtime))
	return &WAZeroRuntime{
		runtime: runtime,
		l:       &sync.RWMutex{},
		codes:   map[string]wazero.CompiledModule{},
	}
}

func (w *WAZeroRuntime) Load(path string) (module wazero.CompiledModule, err error) {
	defer err2.Return(&err)

	w.l.RLock()
	module, ok := w.codes[path]
	if ok {
		w.l.RUnlock()
		return
	}
	w.l.RUnlock()

	w.l.Lock()
	defer w.l.Unlock()

	b := try.To1(os.ReadFile(path))
	ctx := context.Background()
	module = try.To1(w.runtime.CompileModule(ctx, b, wazero.NewCompileConfig()))
	w.codes[path] = module
	return
}

func (w *WAZeroRuntime) Run(path string, config wazero.ModuleConfig) (err error) {
	defer err2.Return(&err)

	code := try.To1(w.Load(path))
	ctx := context.Background()
	id := xid.New().String()
	config = config.WithName(id)
	executedModule := try.To1(w.runtime.InstantiateModule(ctx, code, config))
	executedModule.Close(ctx)
	return
}

func (w *WAZeroRuntime) Unload(path string) (err error) {
	return
}
