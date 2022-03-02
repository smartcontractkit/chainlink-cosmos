package terra

import (
	"bytes"
	"fmt"
	"math/big"
)

func ToInt(s []byte, numBytes uint) (*big.Int, error) {
	if uint(len(s)) != numBytes {
		return nil, fmt.Errorf("invalid int length: expected %d got %d", numBytes, len(s))
	}
	val := (&big.Int{}).SetBytes(s)
	numBits := numBytes * 8
	// 2**(numBits-1) - 1
	maxPositive := big.NewInt(0).Sub(big.NewInt(0).Lsh(big.NewInt(1), numBits-1), big.NewInt(1))
	negative := val.Cmp(maxPositive) > 0
	if negative {
		// Get the complement wrt to 2^numBits
		maxUint := big.NewInt(1)
		maxUint.Lsh(maxUint, numBits)
		val.Sub(maxUint, val)
		val.Neg(val)
	}
	return val, nil
}

func ToBytes(o *big.Int, numBytes uint) ([]byte, error) {
	negative := o.Sign() < 0
	val := (&big.Int{})
	numBits := numBytes * 8
	if negative {
		// compute two's complement as 2**numBits - abs(o) = 2**numBits + o
		val.SetInt64(1)
		val.Lsh(val, numBits)
		val.Add(val, o)
	} else {
		val.Set(o)
	}
	b := val.Bytes() // big-endian representation of abs(val)
	if uint(len(b)) > numBytes {
		return nil, fmt.Errorf("b must fit in %v bytes", numBytes)
	}
	b = bytes.Join([][]byte{bytes.Repeat([]byte{0}, int(numBytes)-len(b)), b}, []byte{})
	if uint(len(b)) != numBytes {
		return nil, fmt.Errorf("wrong length; there must be an error in the padding of b: %v", b)
	}
	return b, nil
}
