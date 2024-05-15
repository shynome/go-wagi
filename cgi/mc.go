package cgi

import (
	"crypto/rand"

	"github.com/tetratelabs/wazero"
)

func WithCommonConfig(mc wazero.ModuleConfig) wazero.ModuleConfig {
	return mc.
		WithRandSource(rand.Reader).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime()

}
