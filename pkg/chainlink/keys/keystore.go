package keystore

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/terrakey"
)

//go:generate mockery --name Keystore --output ./mocks/ --case=underscore --filename keystore.go

type Keystore interface {
	Get(id string) (terrakey.Key, error)
	GetAll() ([]terrakey.Key, error)
	Create() (terrakey.Key, error)
	Add(key terrakey.Key) error
	Delete(id string) (terrakey.Key, error)
	Import(keyJSON []byte, password string) (terrakey.Key, error)
	Export(id string, password string) ([]byte, error)
	EnsureKey() error
}
