package wagi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/tetratelabs/wazero"
)

type Item struct {
	locker   *sync.RWMutex
	filepath string
	modTime  time.Time
	compiled wazero.CompiledModule
	err      error
	gowasm   bool
}

func NewItem(path string) *Item {
	return &Item{
		locker:   &sync.RWMutex{},
		filepath: path,
	}
}

var (
	ErrWasm          = errors.New("wasm err")
	ErrWasmPathIsDir = fmt.Errorf("the wasm path is dir. %w", ErrWasm)
)

func (s *Item) Init(rt wazero.Runtime) {
	defer err2.Catch(func(err error) { s.err = err })

	f := try.To1(os.Open(s.filepath))
	defer f.Close()
	{ // file info
		info := try.To1(f.Stat())
		if info.IsDir() {
			try.To(ErrWasmPathIsDir)
		}
		s.modTime = info.ModTime()
	}
	b := try.To1(io.ReadAll(f))

	ctx := context.Background()
	s.compiled = try.To1(rt.CompileModule(ctx, b))

	for _, f := range s.compiled.ImportedFunctions() {
		switch n, _, _ := f.Import(); n {
		case "go":
			s.gowasm = true
			break
		}
	}
}

func (s *Item) Error() error {
	s.locker.RLock()
	defer s.locker.RUnlock()
	return s.err
}

type Items struct {
	wazero.Runtime
}

func (items Items) Load(c *ttlcache.Cache[string, *Item], key string) *ttlcache.Item[string, *Item] {
	item := NewItem(key)
	item.locker.Lock()
	go func() {
		defer item.locker.Unlock()
		item.Init(items.Runtime)
	}()
	_item := c.Set(key, item, 0)
	return _item
}
