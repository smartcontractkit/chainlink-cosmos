package txm

import (
	"context"

	"golang.org/x/exp/maps"
)

func (ka *keystoreAdapter) Accounts(ctx context.Context) ([]string, error) {
	ka.mutex.Lock()
	defer ka.mutex.Unlock()
	err := ka.updateMappingLocked(ctx)
	if err != nil {
		return nil, err
	}
	addresses := maps.Keys(ka.addressToPubKey)

	return addresses, nil
}
