// Package main provides centralized HTTP error response helpers and logging.
package main

import "net/http"

// logError logs an error along with contextual request metadata.
func (app *application) logError(r *http.Request, err error) {
	app.logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	})
}

// errorResponse writes a standardized JSON error message payload with an HTTP status code.
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := envelope{"error": message}

	// Write JSON error; log and send 500 on write failure
	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// serverErrorResponse logs unexpected errors and sends a 500 Internal Server Error response.
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	app.errorResponse(w, r, http.StatusInternalServerError, "the server encountered a problem and could not process your request")
}

// notFoundResponse sends a 404 Not Found JSON error.
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusNotFound, "the requested resource could not be found")
}

// methodNotAllowedResponse sends a 405 Method Not Allowed JSON error.
func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := "the " + r.Method + " method is not supported for this resource"
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

// badRequestResponse sends a 400 Bad Request JSON error with error details.
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

// failedValidationResponse sends a 422 Unprocessable Entity response containing validation errors.
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

// rateLimitExceededResponse sends a 429 Too Many Requests JSON error.
func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusTooManyRequests, "rate limit exceeded")
}
