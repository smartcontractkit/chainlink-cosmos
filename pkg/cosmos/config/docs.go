package config

import (
	_ "embed"

	"github.com/smartcontractkit/chainlink-common/pkg/config/configdoc"
)

//go:embed docs.toml
var docsTOML string

func GenerateDocs() (string, error) {
	//TODO auto-insert newline?
	return configdoc.Generate(docsTOML, "TODO header", "TODO example\n", nil)
}
