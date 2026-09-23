// Package main provides the HTTP application and helper functions used to
// handle JSON requests and responses, as well as background tasks.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// envelope represents the standard top-level JSON response object.
type envelope map[string]any

// writeJSON serializes data as formatted JSON and writes it to the HTTP
// response with the specified status code and additional HTTP headers.
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Write the serialized JSON response body.
	_, err = w.Write(js)
	return err
}

// readJSON decodes a single JSON value from the HTTP request body into dst.
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Limit request bodies to 1 MB
	const maxBytes = 1_048_576

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		// Convert low-level JSON errors into clearer application-level errors.
		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf(
				"body contains badly-formed JSON (at character %d)",
				syntaxError.Offset,
			)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf(
					"body contains incorrect JSON type for field %q",
					unmarshalTypeError.Field,
				)
			}

			return fmt.Errorf(
				"body contains incorrect JSON type (at character %d)",
				unmarshalTypeError.Offset,
			)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(
				err.Error(),
				"json: unknown field ",
			)

			return fmt.Errorf("body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf(
				"body must not be larger than %d bytes",
				maxBytesError.Limit,
			)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// background runs fn in a separate goroutine and tracks it using the
// application's WaitGroup.
func (app *application) background(fn func()) {
	app.wg.Add(1)

	go func() {
		// Mark the background task as finished when the goroutine exits.
		defer app.wg.Done()

		// Recover from panics
		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%v", err), nil)
			}
		}()

		fn()
	}()
}
