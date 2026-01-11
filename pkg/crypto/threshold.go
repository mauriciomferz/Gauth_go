package crypto

// Prototype threshold secret sharing (Shamir) for Ed25519/Future FROST integration.
// This is NOT a full threshold signing protocol; we split the private key seed
// and later reconstruct it to perform a normal signature. Future work replaces
// this with FROST-style distributed key generation + nonce sharing.
//
// Field: Original implementation used a prime field GF(257) which introduced an
// edge case where polynomial evaluations could produce the value 256 that was
// lossy when truncated to a byte, leading to occasional reconstruction failures
// (seed mismatch). We now use GF(2^8) with the AES irreducible polynomial 0x11b.
// All operations are performed in the full byte field ensuring no overflow or
// truncation ambiguity. This eliminates the nondeterministic test failures.

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
)

// Share represents one Shamir share (index + byte slice value).
type Share struct {
	Index int    // x-coordinate (1..n)
	Value []byte // y-values for each secret byte position
}

var (
	ErrThresholdParams = errors.New("invalid_threshold_params")
	ErrReconstruction  = errors.New("reconstruction_failed")
	ErrDuplicateIndex  = errors.New("duplicate_share_index")
	ErrInsufficient    = errors.New("insufficient_shares")
	ErrIndexOutOfRange = errors.New("share_index_out_of_range")
)

// GF(256) tables for multiplication & inversion using polynomial 0x11b.
var gfExp [512]byte // doubled for easy modular reduction
var gfLog [256]byte // log[0] unused

//nolint:gochecknoinits
func init() {
	// Build exp/log tables; generator 0x03 works for AES field
	x := byte(1)
	for i := 0; i < 255; i++ {
		gfExp[i] = x
		gfLog[x] = byte(i)
		// multiply by generator (0x03): x = x * 0x03 in GF(256)
		x = gfMulRaw(x, 0x03)
	}
	// duplicate for wrap-around
	for i := 255; i < 512; i++ {
		gfExp[i] = gfExp[i-255]
	}
}

// Raw GF(256) multiplication (slow) using shift+conditional XOR for table build.
func gfMulRaw(a, b byte) byte {
	var res byte
	for i := 0; i < 8; i++ {
		if (b & 0x01) != 0 {
			res ^= a
		}
		hi := a & 0x80
		a <<= 1
		if hi != 0 {
			a ^= 0x1b
		} // AES polynomial (x^8 + x^4 + x^3 + x + 1)
		b >>= 1
	}
	return res
}

func gfAdd(a, b byte) byte { return a ^ b } // addition/subtraction are XOR

func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	la := gfLog[a]
	lb := gfLog[b]
	return gfExp[int(la)+int(lb)]
}

func gfInv(a byte) (byte, bool) {
	if a == 0 {
		return 0, false
	}
	return gfExp[255-int(gfLog[a])], true // a^(254) = a^{-1}
}

// SplitSecret splits a secret into n shares with threshold t.
// Requirements: n >= t >= 2; secret length > 0.
func SplitSecret(secret []byte, n, t int, rnd io.Reader) ([]Share, error) {
	if rnd == nil {
		rnd = rand.Reader
	}
	if n < t || t < 2 || n < 2 || len(secret) == 0 || n > 255 {
		return nil, ErrThresholdParams
	}
	shares := make([]Share, n)
	// Coefficients per byte: a0=secret byte; a1..a_{t-1} random bytes.
	coeffs := make([][]byte, len(secret))
	for i, sb := range secret {
		row := make([]byte, t)
		row[0] = sb
		if t > 1 {
			if _, err := io.ReadFull(rnd, row[1:]); err != nil {
				return nil, err
			}
		}
		coeffs[i] = row
	}
	// Evaluate f(x) for x=1..n in GF(256)
	for x := 1; x <= n; x++ {
		xx := byte(x)
		val := make([]byte, len(secret))
		for bi := range secret {
			cs := coeffs[bi]
			y := cs[0]
			xp := xx // current power of x (x^1)
			for ci := 1; ci < len(cs); ci++ {
				term := gfMul(cs[ci], xp)
				y = gfAdd(y, term)
				xp = gfMul(xp, xx) // advance power
			}
			val[bi] = y
		}
		shares[x-1] = Share{Index: x, Value: val}
	}
	return shares, nil
}

// Reconstruct recovers the secret from >= t shares.
func Reconstruct(shares []Share, t int) ([]byte, error) {
	if len(shares) < t || t < 2 {
		return nil, ErrInsufficient
	}
	used := map[int]struct{}{}
	valueLen := len(shares[0].Value)
	for _, s := range shares {
		if s.Index < 1 || s.Index > 255 {
			return nil, ErrIndexOutOfRange
		}
		if len(s.Value) != valueLen {
			return nil, ErrReconstruction
		}
		if _, exists := used[s.Index]; exists {
			return nil, ErrDuplicateIndex
		}
		used[s.Index] = struct{}{}
	}
	subset := shares[:t]
	secret := make([]byte, valueLen)
	for bi := 0; bi < valueLen; bi++ {
		var total byte = 0
		for i, si := range subset {
			xi := byte(si.Index)
			yi := si.Value[bi]
			num := byte(1)
			den := byte(1)
			for j, sj := range subset {
				if i == j {
					continue
				}
				xj := byte(sj.Index)
				// num *= x_j ; den *= (x_j + x_i)
				num = gfMul(num, xj)
				den = gfMul(den, gfAdd(xj, xi)) // xi - xj == xi + xj in GF(256)
			}
			invDen, ok := gfInv(den)
			if !ok {
				return nil, ErrReconstruction
			}
			li0 := gfMul(num, invDen)
			contrib := gfMul(yi, li0)
			total = gfAdd(total, contrib)
		}
		secret[bi] = total
	}
	return secret, nil
}

// modInverse computes modular inverse using extended Euclid.
// (modInverse removed; GF(256) inversion handled by tables)

// ThresholdSignEd25519 reconstructs an Ed25519 private key seed from shares and produces a signature.
// We expect the original public key for reconstruction of full private key (seed || public).
func ThresholdSignEd25519(shares []Share, t int, public ed25519.PublicKey, msg []byte) ([]byte, error) {
	seed, err := Reconstruct(shares, t)
	if err != nil {
		return nil, err
	}
	if len(public) != ed25519.PublicKeySize {
		return nil, errors.New("invalid_public_key")
	}
	if len(seed) != ed25519.SeedSize {
		return nil, errors.New("invalid_seed_size")
	}
	priv := ed25519.NewKeyFromSeed(seed)
	// Ensure derived public matches provided (defense against corrupted shares)
	derived := priv.Public().(ed25519.PublicKey)
	if !ed25519Equal(derived, public) {
		return nil, errors.New("public_mismatch_after_reconstruct")
	}
	return ed25519.Sign(priv, msg), nil
}

func ed25519Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
