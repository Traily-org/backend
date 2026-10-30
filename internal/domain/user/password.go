package user

import "golang.org/x/crypto/bcrypt"

// BcryptCost is the bcrypt work factor used to hash user passwords.
// The project security policy requires a minimum cost of 12.
const BcryptCost = 12

// MaxPasswordBytes is the longest password bcrypt accepts, in bytes.
const MaxPasswordBytes = 72

// HashPassword returns the bcrypt hash of plain using BcryptCost.
// It returns ErrPasswordTooLong if plain exceeds MaxPasswordBytes.
func HashPassword(plain string) (string, error) {
	if len(plain) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// CheckPassword reports whether plain matches the bcrypt hash.
// It returns false if hash is malformed or does not match.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
