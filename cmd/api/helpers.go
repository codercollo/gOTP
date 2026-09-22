// Package main provides JSON encoding/decoding utilities and async task management helpers.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// envelope wraps response payloads in a top-level JSON object.
type envelope map[string]any

// writeJSON serializes data to indented JSON and writes it with HTTP headers and status code.
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// Marshal envelope payload
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	// Set custom and default HTTP headers
	for key, value := range headers {
		w.Header()[key] = value
	}
	w.Header().Set("Content-Type", "application/json")

	// Write status and response body
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

// readJSON decodes request JSON into dst, enforcing a 1 MB max body limit and strict parsing rules.
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Restrict payload size to 1MB
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Configure strict decoder
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode JSON and map error types to user-friendly messages
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrent JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body conrtains unknown key %s", &fieldName)
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must be larger than %d bytes", maxBytes)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err

		}
	}

	// Verify no trailing JSON objects exist
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil

}

// background executes fn in a panic-recovered gorutine tracked for graceful shutdown
func (app *application) background(fn func()) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()

		// Recover from panic
		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%v", err), nil)
			}
		}()

		fn()
	}()
}
