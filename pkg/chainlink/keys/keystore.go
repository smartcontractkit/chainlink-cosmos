package keys

import (
	"fmt"

	"github.com/pkg/errors"
)

//go:generate mockery --name Keystore --output ./mocks/ --case=underscore --filename keystore.go

type Keystore interface {
	Get(id string) (Key, error)
	GetAll() ([]Key, error)
	Create() (Key, error)
	Add(key Key) error
	Delete(id string) (Key, error)
	Import(keyJSON []byte, password string) (Key, error)
	Export(id string, password string) ([]byte, error)
	EnsureKey() error
}
