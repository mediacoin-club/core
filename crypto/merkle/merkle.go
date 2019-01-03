package merkle

import (
	"bytes"
	"crypto/sha256"
)

const HashSize = 256 / 8 // hash size

func hash(a, b []byte) []byte {
	h := sha256.New()
	h.Write(a)
	h.Write(b)
	return h.Sum(nil)
}

func Root(hashes ...[]byte) []byte {
	if len(hashes) == 0 {
		return nil
	}
	for len(hashes) > 1 {
		n := len(hashes)
		m := n / 2
		for j := 0; j < m; j++ {
			hashes[j] = hash(hashes[j*2], hashes[j*2+1])
		}
		if n%2 != 0 {
			hashes[m] = hashes[n-1]
			m++
		}
		hashes = hashes[:m]
	}
	return hashes[0]
}

func Proof(hashes [][]byte, i int) (proof, root []byte) {
	if len(hashes) == 0 {
		return
	}
	for len(hashes) > 1 {
		n := len(hashes)
		m := n / 2
		if i < m*2 {
			if i%2 == 0 {
				proof = append(proof, 0)
				proof = append(proof, hashes[i+1]...)
			} else {
				proof = append(proof, 1)
				proof = append(proof, hashes[i-1]...)
			}
		}
		i /= 2
		for j := 0; j < m; j++ {
			hashes[j] = hash(hashes[j*2], hashes[j*2+1])
		}
		if n%2 != 0 {
			hashes[m] = hashes[n-1]
			m++
		}
		hashes = hashes[:m]
	}
	root = hashes[0]
	return
}

func ProofRoot(key, proof []byte) (root []byte) {
	for len(proof) > 0 {
		if len(proof) < HashSize+1 {
			return nil
		}
		if proof[0] == 0 {
			key = hash(key, proof[1:HashSize+1])
		} else {
			key = hash(proof[1:HashSize+1], key)
		}
		proof = proof[HashSize+1:]
	}
	return key
}

func Verify(key, proof, root []byte) bool {
	r := ProofRoot(key, proof)
	return r != nil && bytes.Equal(r, root)
}
