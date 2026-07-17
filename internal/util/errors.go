package util

import "errors"

const ErrorDeleteNotAllowed = 10
const ErrorBadParams = 20
const ErrorFailedConsulConnection = 30
const ErrorFailedConsulTxn = 31
const ErrorFailedReadingResponse = 40
const ErrorFailedJsonEncode = 50
const ErrorFailedJsonDecode = 51
const ErrorFailedCloning = 60
const ErrorFailedMustache = 70
const ErrorFailedHTTPServer = 80

type GonsulError struct {
	Code int
	Err  error
}

func (e GonsulError) Error() string {
	if e.Err == nil {
		return ""
	}

	return e.Err.Error()
}

func (e GonsulError) Unwrap() error {
	return e.Err
}

func NewGonsulError(err error, errorCode int) GonsulError {
	return GonsulError{Code: errorCode, Err: err}
}

func ErrorCode(err error, fallback int) int {
	var gonsulErr GonsulError
	if errors.As(err, &gonsulErr) {
		return gonsulErr.Code
	}

	return fallback
}
