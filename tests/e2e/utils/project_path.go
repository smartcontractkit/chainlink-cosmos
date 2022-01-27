package utils

import (
	"path/filepath"
	"runtime"
)

var (
	_, b, _, _ = runtime.Caller(0)
	// ProjectRoot Root folder of this project
	ProjectRoot = filepath.Join(filepath.Dir(b), "/../../..")
	// ContractsDir contracts dir with wasm artifacts
	ContractsDir = filepath.Join(ProjectRoot, "artifacts")
	// CommonContractsDir is common artifacts dir, for example cw20_base.wasm
	CommonContractsDir = filepath.Join(TestsDir, "common_artifacts")
	// TestsDir path to e2e tests dir
	TestsDir = filepath.Join(ProjectRoot, "tests", "e2e")
)
