package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
)

// FriendKeyChecker is an interface for checking whether a friend key already
// exists. This abstraction lets GenerateFriendKey work against a real database
// or a test stub without importing database-specific packages.
type FriendKeyChecker interface {
	FriendKeyExists(key string) (bool, error)
}

// GenerateFriendKey produces a human-readable, verbally shareable session join
// key in the format "adjective-noun-number" (e.g. "happy-tiger-42"). Words are
// drawn from petname's built-in adjective and noun pools using its internal
// random selection. The number is in the range 1-99 and uses crypto/rand for
// cryptographically secure generation.
//
// The function retries up to 100 times to find a unique key that does not
// collide with any existing session. If all 100 attempts collide (extremely
// unlikely given the keyspace), it returns an error.
func GenerateFriendKey(checker FriendKeyChecker) (string, error) {
	for attempts := 0; attempts < 100; attempts++ {
		// petname.Generate(2, "-") produces "adjective-noun" from a
		// 37k adjective + 6k noun pool, giving ~220M combinations
		// before the number suffix.
		words := petname.Generate(2, "-")
		num, err := cryptoRandInt(99)
		if err != nil {
			return "", fmt.Errorf("generating friend key number: %w", err)
		}
		num++ // range 1-99
		key := fmt.Sprintf("%s-%d", words, num)

		exists, err := checker.FriendKeyExists(key)
		if err != nil {
			return "", fmt.Errorf("checking friend key existence: %w", err)
		}
		if !exists {
			return key, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique friend key after 100 attempts")
}

// NormalizeFriendKey lowercases and trims whitespace from a friend key so that
// lookups are case-insensitive. Users can type "Happy-Tiger-42" and still match
// the stored key "happy-tiger-42".
func NormalizeFriendKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// cryptoRandInt returns a cryptographically random integer in the range [0, max).
// Uses crypto/rand rather than math/rand to avoid predictable key generation.
func cryptoRandInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
