package syncer

import (
	"errors"
	"net"
	"testing"

	"github.com/pkg/sftp"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		err  error
		kind int
		code int
		text string
	}{
		{"refused", errors.New("dial tcp: connection refused"), KindUnreachable, 2, "unreachable"},
		{"auth", errors.New("ssh: unable to authenticate, attempted methods [publickey]"), KindAuth, 3, "auth failed"},
		{"space", &sftp.StatusError{Code: sshFxNoSpaceOnFilesystem}, KindDiskFull, 4, "disk full"},
		{"quota", errors.New("disk quota exceeded"), KindDiskFull, 4, "disk full"},
		{"hostkey", errors.New("knownhosts: key is unknown"), KindOther, 1, "knownhosts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.err)
			var se *Error
			if !errors.As(got, &se) || se.Kind != tc.kind {
				t.Fatalf("kind: %v", got)
			}
			if ExitCode(got) != tc.code {
				t.Fatalf("exit %d", ExitCode(got))
			}
			if !stringsContains(got.Error(), tc.text) {
				t.Fatalf("text %q", got.Error())
			}
		})
	}
	op := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connect: connection refused")}
	got := Classify(op)
	var se *Error
	if !errors.As(got, &se) || se.Kind != KindUnreachable {
		t.Fatalf("op error: %v", got)
	}
}

func stringsContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && contains(s, sub)))
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
