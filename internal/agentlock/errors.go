package agentlock

import (
	"errors"
	"fmt"
)

type codedError struct {
	code    int
	message string
	err     error
}

func (e *codedError) Error() string {
	if e.err == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %v", e.message, e.err)
}

func (e *codedError) Unwrap() error {
	return e.err
}

func (e *codedError) Code() int {
	return e.code
}

func newUserError(message string, err error) error {
	return &codedError{
		code:    exitCodeUser,
		message: message,
		err:     err,
	}
}

func newSystemError(message string, err error) error {
	return &codedError{
		code:    exitCodeSystem,
		message: message,
		err:     err,
	}
}

func errorCode(err error) int {
	if err == nil {
		return exitCodeSuccess
	}
	var coded interface{ Code() int }
	if errors.As(err, &coded) {
		return coded.Code()
	}
	return exitCodeUser
}
