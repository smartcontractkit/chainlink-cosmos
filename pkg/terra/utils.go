package terra

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"

	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

// HexToByteArray is a wrapper for hex.DecodeString
func HexToByteArray(s string, b *[]byte) (err error) {
	*b, err = hex.DecodeString(s)
	return err
}

// HexToConfigDigest converts a hex string to ConfigDigest
func HexToConfigDigest(s string, digest *types.ConfigDigest) (err error) {
	// parse byte array encoded as hex string
	var byteArr []byte
	if err = HexToByteArray(s, &byteArr); err != nil {
		return
	}

	*digest, err = types.BytesToConfigDigest(byteArr)
	return
}

// HexToArray process a hex encoded array by splitting
// currently not used, but left in case needed in the future
// `n` specifies the expected length of each element
// `output` is the expected output array
// `postprocess` allows the []byte output to be processed in any way
func HexToArray(s string, n int, output interface{}, parse func([]byte) interface{}) error {
	// check to make sure hex encoded 2*n characters
	if len(s)%(n*2) != 0 {
		return errors.New("invalid string length")
	}

	// parse to bytes
	var b []byte
	if err := HexToByteArray(s, &b); err != nil {
		return err
	}

	// create new array of parsed values based on `n` elements
	arr := reflect.ValueOf(output) // get the array
	arr = arr.Elem()               // make settable
	for i := 0; i < len(b); i += n {
		// append values to array + use parse for type conversion
		arr = reflect.Append(arr, reflect.ValueOf(parse(b[i:i+n])))
	}

	// writer
	writer := reflect.ValueOf(output) // create output writer
	writer = writer.Elem()            // make settable
	writer.Set(arr)                   // set
	return nil
}

// RawMessageStringIntToInt converts a json string number to an int
func RawMessageStringIntToInt(msg json.RawMessage) (int, error) {
	var temp string
	if err := json.Unmarshal(msg, &temp); err != nil {
		return 0, err
	}
	return strconv.Atoi(temp)
}

func MustAccAddress(addr string) cosmosSDK.AccAddress {
	accAddr, err := cosmosSDK.AccAddressFromBech32(addr)
	if err != nil {
		panic(err)
	}
	return accAddr
}

const (
	byteWidth128 = 16
	bitWidth128  = byteWidth128 * 8
)

// ContractConfigToOCRConfig converts the output onchain_config to the type
// expected by OCR
func ContractConfigToOCRConfig(in []byte) ([]byte, error) {
	// onchain =              <8bit version><128bit min><128bit max>
	// libocr median plugin = <8bit version><192bit min><192bit max>
	if len(in) != 33 {
		return nil, fmt.Errorf("invalid config length: expected 33 got %d", len(in))
	}
	version := in[0:1]
	min, err := Parse128BitSignedInt(in[1 : byteWidth128+1])
	if err != nil {
		return nil, err
	}
	max, err := Parse128BitSignedInt(in[1+byteWidth128:])
	if err != nil {
		return nil, err
	}
	minBytes, err := median.ToBytes(min)
	if err != nil {
		return nil, err
	}
	maxBytes, err := median.ToBytes(max)
	if err != nil {
		return nil, err
	}
	return bytes.Join([][]byte{version, minBytes, maxBytes}, []byte{}), nil
}

func Parse128BitSignedInt(s []byte) (*big.Int, error) {
	if len(s) != byteWidth128 {
		return nil, fmt.Errorf("invalid int length: expected %d got %d", byteWidth128, len(s))
	}
	val := (&big.Int{}).SetBytes(s)
	// 2**127 - 1
	maxPositive := big.NewInt(0).Sub(big.NewInt(0).Lsh(big.NewInt(1), bitWidth128-1), big.NewInt(1))
	negative := val.Cmp(maxPositive) > 0
	if negative {
		// Get the complement wrt to 2^128
		maxUint := big.NewInt(1)
		maxUint.Lsh(maxUint, bitWidth128)
		val.Sub(maxUint, val)
		val.Neg(val)
	}
	return val, nil
}
