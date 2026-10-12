package access

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// NewTOTPSecret returns a base32 secret for Google Authenticator.
func NewTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "="), nil
}

// VerifyTOTP accepts the current 30-second code and one step either side.
func VerifyTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	raw, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret) + strings.Repeat("=", (8-len(secret)%8)%8))
	if err != nil {
		return false
	}
	counter := uint64(now.Unix() / 30)
	for _, c := range []uint64{counter - 1, counter, counter + 1} {
		if hotp(raw, c) == code {
			return true
		}
	}
	return false
}

func hotp(secret []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", bin%1000000)
}
