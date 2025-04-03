package lib_error

import (
	"fmt"
)

const DefaultErrorCode = 256

type AppError struct {
	Code    int
	Message string
}

func (self *AppError) Error() string {
	return fmt.Sprintf("%s(%v)", self.Message, self.Code)
}

func NewAppError(code int, message string, args ...interface{}) error {
	if 0 < len(args) {
		message = fmt.Sprintf(message, args...)
	}
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func NewAppErrorWithStackTrace(code int, message string, args ...interface{}) error {
	if 0 < len(args) {
		message = fmt.Sprintf(message, args...)
	}
	message = fmt.Sprintf("%s\n%s", message, StackTrace())
	return NewAppError(code, message)
}

func WrapError(err error) error {
	if err == nil {
		return nil
	}
	switch err.(type) {
	case *AppError:
		return err
	}
	return NewAppErrorWithStackTrace(DefaultErrorCode, err.Error())
}
