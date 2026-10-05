// Package random generates reproducible random test data: emails, strings,
// numbers, UUIDs.
//
// Everything is derived from a seed. A run normally uses a fresh seed, so
// repeated runs do not collide on "already exists"; the seed is printed in
// the report, and running again with the same seed reproduces the same data.
//
// A Generator produces a stream of values. To keep data independent of the
// order in which scenarios run, each scenario call gets its own stream with
// Derive: it depends only on the seed and on the key (the place of the call
// in the group tree), not on what other streams have produced. So adding a
// step to one scenario does not change the data of another, an isolated run
// of a scenario gets the same data as a full run, and groups may run in
// parallel.
package random

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand/v2"
)

// Generator is a stream of random values. It is not safe for concurrent use:
// give every goroutine its own stream with Derive.
type Generator struct {
	// id identifies the stream: it is a hash of the seed and of the keys the
	// stream was derived with. The stream state is initialized from it.
	id  [sha256.Size]byte
	rng *rand.Rand
}

// New returns the root stream of a seed.
func New(seed uint64) *Generator {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], seed)
	return newGenerator(sha256.Sum256(buf[:]))
}

func newGenerator(id [sha256.Size]byte) *Generator {
	// PCG is used because its output for a given state is fixed by the Go
	// compatibility promise, unlike the top-level functions of math/rand.
	src := rand.NewPCG(binary.BigEndian.Uint64(id[:8]), binary.BigEndian.Uint64(id[8:16]))
	return &Generator{id: id, rng: rand.New(src)}
}

// Derive returns an independent stream for key. The result depends only on
// the identity of g and on key, not on how many values g has produced, so
// streams can be derived in any order. Deriving with the same key twice
// gives two streams that produce the same values.
//
// The engine derives a stream for every scenario call from the group path
// and the call alias.
func (g *Generator) Derive(key string) *Generator {
	h := sha256.New()
	h.Write(g.id[:])
	h.Write([]byte(key))
	var id [sha256.Size]byte
	h.Sum(id[:0])
	return newGenerator(id)
}

// chars returns a string of n characters drawn from alphabet.
func (g *Generator) chars(alphabet string, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[g.rng.IntN(len(alphabet))]
	}
	return string(b)
}

// between returns an integer in [min, max]. The range may span the whole
// int64, so the width is computed in uint64 to avoid overflow.
func (g *Generator) between(min, max int64) int64 {
	width := uint64(max) - uint64(min) // max >= min, so this does not wrap below zero
	if width == ^uint64(0) {
		return int64(g.rng.Uint64())
	}
	return int64(uint64(min) + g.rng.Uint64N(width+1))
}
