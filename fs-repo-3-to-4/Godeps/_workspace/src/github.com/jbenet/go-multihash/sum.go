package multihash

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
)

var ErrSumNotSupported = errors.New("Function not implemented. Complain to lib maintainer.")

func Sum(data []byte, code int, length int) (Multihash, error) {
	m := Multihash{}
	err := error(nil)
	if !ValidCode(code) {
		return m, fmt.Errorf("invalid multihash code %d", code)
	}

	var d []byte
	switch code {
	case SHA1:
		d = sumSHA1(data)
	case SHA2_256:
		d = sumSHA256(data)
	case SHA2_512:
		d = sumSHA512(data)
	default:
		return m, ErrSumNotSupported
	}

	if err != nil {
		return m, err
	}

	if length < 0 {
		var ok bool
		length, ok = DefaultLengths[code]
		if !ok {
			return m, fmt.Errorf("no default length for code %d", code)
		}
	}

	return Encode(d[0:length], code)
}

func sumSHA1(data []byte) []byte {
	a := sha1.Sum(data)
	return a[0:20]
}

func sumSHA256(data []byte) []byte {
	a := sha256.Sum256(data)
	return a[0:32]
}

func sumSHA512(data []byte) []byte {
	a := sha512.Sum512(data)
	return a[0:64]
}
