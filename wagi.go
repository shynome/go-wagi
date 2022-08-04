package wagi

import (
	"context"
	"os"
	"sync"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/wasi_snapshot_preview1"
)

type Wagi struct {
	l       *sync.RWMutex
	runtime wazero.Runtime
}

func NewWagi() *Wagi {
	ctx := context.Background()
	runtime := wazero.NewRuntimeWithConfig(wazero.NewRuntimeConfig().WithWasmCore2())
	try.To1(wasi_snapshot_preview1.Instantiate(ctx, runtime))
	return &Wagi{
		runtime: runtime,
		l:       &sync.RWMutex{},
	}
}

func (w *Wagi) Load(path string) (module wazero.CompiledModule, err error) {
	defer err2.Return(&err)
	b := try.To1(os.ReadFile(path))
	ctx := context.Background()
	module = try.To1(w.runtime.CompileModule(ctx, b, wazero.NewCompileConfig()))
	return
}

func (w *Wagi) Run(module wazero.CompiledModule, config wazero.ModuleConfig) (m api.Module, err error) {
	defer err2.Return(&err)
	ctx := context.Background()
	m = try.To1(w.runtime.InstantiateModule(ctx, module, config))
	return
}
