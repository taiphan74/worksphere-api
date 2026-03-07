package errors

import stderrors "errors"

type AppError struct {
	StatusCode int
	Code       string
	Message    string
	Err        error
}

func New(statusCode int, code, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

func Wrap(statusCode int, code, message string, err error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Err:        err,
	}
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

func As(err error, target any) bool {
	return stderrors.As(err, target)
}
