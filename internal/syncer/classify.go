package syncer

import (
	"errors"
	"net"
	"strings"

	"github.com/pkg/sftp"
)

// SSH_FX_NO_SPACE_ON_FILESYSTEM from the SFTP spec. pkg/sftp does not export it.
const sshFxNoSpaceOnFilesystem = 14

const (
	KindOther = iota
	KindUnreachable
	KindAuth
	KindDiskFull
)

type Error struct {
	Kind int
	Err  error
}

func (e *Error) Error() string {
	msg := ""
	if e.Err != nil {
		msg = e.Err.Error()
	}
	switch e.Kind {
	case KindUnreachable:
		return "unreachable: " + msg
	case KindAuth:
		return "auth failed: " + msg
	case KindDiskFull:
		return "disk full: " + msg
	default:
		return msg
	}
}

func (e *Error) Unwrap() error { return e.Err }

func ExitCode(err error) int {
	var se *Error
	if errors.As(err, &se) {
		switch se.Kind {
		case KindUnreachable:
			return 2
		case KindAuth:
			return 3
		case KindDiskFull:
			return 4
		}
	}
	return 1
}

func Classify(err error) error {
	if err == nil {
		return nil
	}
	var already *Error
	if errors.As(err, &already) {
		return already
	}
	if diskFull(err) {
		return &Error{Kind: KindDiskFull, Err: err}
	}
	msg := err.Error()
	if strings.Contains(msg, "knownhosts:") || strings.Contains(msg, "host key") {
		return &Error{Kind: KindOther, Err: err}
	}
	if strings.Contains(msg, "unable to authenticate") || strings.Contains(msg, "no supported methods") {
		return &Error{Kind: KindAuth, Err: err}
	}
	var op *net.OpError
	if errors.As(err, &op) {
		return &Error{Kind: KindUnreachable, Err: err}
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "no route to host") ||
		strings.Contains(lower, "network is unreachable") ||
		strings.Contains(lower, "connection timed out") ||
		strings.Contains(lower, "no such host") {
		return &Error{Kind: KindUnreachable, Err: err}
	}
	return &Error{Kind: KindOther, Err: err}
}

func diskFull(err error) bool {
	var st *sftp.StatusError
	if errors.As(err, &st) && st.Code == sshFxNoSpaceOnFilesystem {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "no space left") || strings.Contains(lower, "disk quota exceeded")
}
