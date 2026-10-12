package access

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

func TestTOTP(t *testing.T) {
	secret, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret) + strings.Repeat("=", (8-len(secret)%8)%8))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	code := hotp(raw, uint64(now.Unix()/30))
	if !VerifyTOTP(secret, code, now) {
		t.Fatal("expected current code")
	}
	if code != "000000" && VerifyTOTP(secret, "000000", now) {
		t.Fatal("unexpected match")
	}
}
