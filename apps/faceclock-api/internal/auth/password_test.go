package auth

import (
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	password := "SecretPass123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	match, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("unexpected error verifying password: %v", err)
	}
	if !match {
		t.Errorf("expected password to match")
	}

	matchWrong, err := VerifyPassword("WrongPass456", hash)
	if err != nil {
		t.Fatalf("unexpected error verifying wrong password: %v", err)
	}
	if matchWrong {
		t.Errorf("expected wrong password not to match")
	}

	// Verify salt randomness: same password produces different hashes
	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error hashing second time: %v", err)
	}
	if hash == hash2 {
		t.Errorf("expected different hashes due to random salt, got identical hashes")
	}
	match2, err := VerifyPassword(password, hash2)
	if err != nil || !match2 {
		t.Errorf("expected second hash to verify successfully")
	}
}

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		name      string
		password  string
		minLen    int
		expectErr bool
	}{
		{"valid strong password", "StrongPass123", 10, false},
		{"too short", "Pass1", 10, true},
		{"only letters", "PasswordNoNumbers", 10, true},
		{"only numbers", "123456789012", 10, true},
		{"weak password in dictionary", "faceclock123", 10, true},
		{"weak password uppercase", "Password123", 10, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.password, tc.minLen)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for %q, got nil", tc.password)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("expected no error for %q, got: %v", tc.password, err)
			}
		})
	}
}

func TestGenerateTemporaryPassword(t *testing.T) {
	for i := 0; i < 5; i++ {
		pwd, err := GenerateTemporaryPassword()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pwd) != 12 {
			t.Errorf("expected length 12, got %d", len(pwd))
		}
		if err := ValidatePassword(pwd, 10); err != nil {
			t.Errorf("generated password failed validation: %v", err)
		}
	}
}
