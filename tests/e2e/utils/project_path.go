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
	// CodeIds path to the codeIds directory
	CodeIds = filepath.Join(ProjectRoot, "packages-ts", "gauntlet-terra-contracts", "codeIds")
	// Reports path to the gauntlet reports directory
	Reports = filepath.Join(ProjectRoot, "tests", "e2e", "smoke", "reports")
	// Rdd path to the gauntlet rdd directory
	Rdd = filepath.Join(ProjectRoot, "tests", "e2e", "smoke", "rdd")
	// Networks path to the networks directory
	Networks = filepath.Join(ProjectRoot, "packages-ts", "gauntlet-terra-contracts", "networks")
)
