package user

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword_ValidPassword_ReturnsBcryptHashWithProjectCost(t *testing.T) {
	hash, err := HashPassword("supersecret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "supersecret" {
		t.Fatal("HashPassword() returned the plain password")
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost() error = %v", err)
	}
	if cost != BcryptCost {
		t.Fatalf("bcrypt cost = %d, want %d", cost, BcryptCost)
	}
}

func TestHashPassword_SamePasswordTwice_ReturnsDifferentHashes(t *testing.T) {
	first, err := HashPassword("supersecret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	second, err := HashPassword("supersecret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if first == second {
		t.Fatal("HashPassword() returned identical hashes, want salted hashes")
	}
}

func TestHashPassword_TooLongPassword_ReturnsErrPasswordTooLong(t *testing.T) {
	if _, err := HashPassword(strings.Repeat("a", 73)); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("HashPassword() error = %v, want %v", err, ErrPasswordTooLong)
	}
}

func TestHashPassword_MultiBytePasswordOverLimit_ReturnsErrPasswordTooLong(t *testing.T) {
	// 40 characters but 80 bytes in UTF-8.
	if _, err := HashPassword(strings.Repeat("é", 40)); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("HashPassword() error = %v, want %v", err, ErrPasswordTooLong)
	}
}

func TestBcryptCost_Value_IsAtLeastTwelve(t *testing.T) {
	if BcryptCost < 12 {
		t.Fatalf("BcryptCost = %d, want >= 12", BcryptCost)
	}
}

func TestCheckPassword_MatchingPassword_ReturnsTrue(t *testing.T) {
	hash, err := HashPassword("supersecret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !CheckPassword(hash, "supersecret") {
		t.Fatal("CheckPassword() = false, want true")
	}
}

func TestCheckPassword_WrongPassword_ReturnsFalse(t *testing.T) {
	hash, err := HashPassword("supersecret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Fatal("CheckPassword() = true, want false")
	}
}

func TestCheckPassword_MalformedHash_ReturnsFalse(t *testing.T) {
	if CheckPassword("not-a-bcrypt-hash", "supersecret") {
		t.Fatal("CheckPassword() = true, want false")
	}
}
