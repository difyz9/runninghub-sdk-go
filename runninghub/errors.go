package runninghub

import (
	"errors"
	"fmt"
)

var ErrNilClient = errors.New("runninghub: nil http client")

type APIError struct {
	Code    int
	Message string
	Details any
}

func (e *APIError) Error() string {
	if e == nil {
		return "runninghub: <nil>"
	}
	if e.Code == 0 {
		return fmt.Sprintf("runninghub: api error: %s", e.Message)
	}
	return fmt.Sprintf("runninghub: api error: code=%d message=%s", e.Code, e.Message)
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	if e == nil {
		return "runninghub: <nil>"
	}
	if e.Body == "" {
		return fmt.Sprintf("runninghub: http error: status=%d", e.StatusCode)
	}
	return fmt.Sprintf("runninghub: http error: status=%d body=%s", e.StatusCode, e.Body)
}
