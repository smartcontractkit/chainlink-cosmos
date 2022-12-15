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
	// TestsDir path to tests dir
	TestsDir = filepath.Join(ProjectRoot, "tests")
	// CommonContractsDir is common artifacts dir, for example cw20_base.wasm
	CommonContractsDir = filepath.Join(TestsDir, "common_artifacts")
	// Reports path to the gauntlet reports directory
	Reports = filepath.Join(TestsDir, "smoke", "reports")
	// Rdd path to the gauntlet rdd directory
	Rdd = filepath.Join(TestsDir, "smoke", "rdd")
	// GauntletTerraContracts path to the gauntlet-terra-contracts dir
	GauntletTerraContracts = filepath.Join(ProjectRoot, "packages-ts", "gauntlet-terra-contracts")
	// Networks path to the networks directory
	Networks = filepath.Join(GauntletTerraContracts, "networks")
	// CodeIds path to the codeIds directory
	CodeIds = filepath.Join(GauntletTerraContracts, "codeIds")
)
