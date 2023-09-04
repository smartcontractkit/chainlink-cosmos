package testutil

import (
	"errors"

	injhd "github.com/InjectiveLabs/sdk-go/chain/crypto/hd"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bip39 "github.com/cosmos/go-bip39"
)

// CreateMnemonic - create new mnemonic
func CreateMnemonic() (string, error) {
	// Default number of words (24): This generates a mnemonic directly from the
	// number of words by reading system entropy.
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}

	return bip39.NewMnemonic(entropy)
}

// TODO: these should be configurable
var (
	// ATOM coin type
	// ref: https://github.com/satoshilabs/slips/blob/master/slip-0044.md
	// coinType              uint32 = 118
	defaultDerivationPath string = "m/44'/60'/0'/0/0"
)

// CreateHDPath returns BIP 44 object from account and index parameters.
func CreateHDPath(account uint32, index uint32) string {
	return defaultDerivationPath
	// return hd.CreateHDPath(coinType, account, index).String()
}

func CreateKeyFromMnemonic(mnemonic string) (cryptotypes.PrivKey, sdk.AccAddress, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, nil, errors.New("invalid mnemonic")
	}

	// algo, err := keyring.NewSigningAlgoFromString(string(hd.Secp256k1Type), keyring.SigningAlgoList{hd.Secp256k1})
	// if err != nil {
	// 	return nil, nil, err
	// }
	algo := injhd.EthSecp256k1

	// create master key and derive first key for keyring
	hdPath := CreateHDPath(0, 0)
	bz, err := algo.Derive()(mnemonic, "", hdPath)
	if err != nil {
		return nil, nil, err
	}

	privKey := algo.Generate()(bz)
	addr := sdk.AccAddress(privKey.PubKey().Address())
	return privKey, addr, nil
}
