package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxRequestBodyBytes int64 = 1 << 20 // 1 MiB

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError
		var maxBytesErr *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("invalid json")
		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("invalid json")
		case errors.As(err, &unmarshalTypeErr):
			return fmt.Errorf("invalid value for field %q", unmarshalTypeErr.Field)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			field := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("unknown field %s", field)
		case errors.Is(err, io.EOF):
			return fmt.Errorf("request body is empty")
		case errors.As(err, &maxBytesErr):
			return fmt.Errorf("request body too large")
		default:
			return fmt.Errorf("invalid json")
		}
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain a single JSON object")
	}

	return nil
}
