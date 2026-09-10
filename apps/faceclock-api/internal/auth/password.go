package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 2
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

// DummyHash is computed at init time to provide constant-time password verification
// when a user lookup fails, preventing timing side-channel attacks on user enumeration.
var DummyHash string

func init() {
	var err error
	DummyHash, err = HashPassword("dummy-password-for-timing-mitigation-123456")
	if err != nil {
		panic(fmt.Sprintf("failed to initialize dummy password hash: %v", err))
	}
}

// HashPassword hashes a plain-text password using Argon2id with standard parameters.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads, b64Salt, b64Hash)

	return encoded, nil
}

// VerifyPassword checks whether a plain-text password matches an Argon2id encoded hash.
// It uses subtle.ConstantTimeCompare to avoid timing attacks.
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return false, errors.New("unsupported hash algorithm")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("parsing version: %w", err)
	}
	if version != argon2.Version {
		return false, errors.New("incompatible argon2 version")
	}

	var memory uint32
	var iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return false, fmt.Errorf("parsing argon2 parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decoding salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decoding hash: %w", err)
	}

	comparisonHash := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(expectedHash)))

	if subtle.ConstantTimeCompare(expectedHash, comparisonHash) == 1 {
		return true, nil
	}

	return false, nil
}

// ValidatePassword ensures password meets minimum requirements:
// - at least minLength characters (default 10 if minLength <= 0)
// - contains at least one letter and at least one digit
// - not in the weak passwords dictionary
func ValidatePassword(password string, minLength int) error {
	if minLength <= 0 {
		minLength = 10
	}

	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters", minLength)
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else if unicode.IsDigit(r) {
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return errors.New("password must contain both letters and numbers")
	}

	if IsWeakPassword(password) {
		return errors.New("password is too common or easily guessed")
	}

	return nil
}

// unambiguousChars avoids visually confusing characters like 0/O and 1/l/I.
const unambiguousChars = "23456789abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

// GenerateTemporaryPassword produces a 12-character cryptographically secure temporary password.
func GenerateTemporaryPassword() (string, error) {
	const length = 12
	charsLen := big.NewInt(int64(len(unambiguousChars)))

	for {
		b := make([]byte, length)
		for i := 0; i < length; i++ {
			num, err := rand.Int(rand.Reader, charsLen)
			if err != nil {
				return "", fmt.Errorf("generating random character: %w", err)
			}
			b[i] = unambiguousChars[num.Int64()]
		}

		pwd := string(b)
		// Ensure it passes validation (has letter, digit, not weak)
		if err := ValidatePassword(pwd, 10); err == nil {
			return pwd, nil
		}
	}
}
