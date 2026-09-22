// Package main provides the HTTP handlers, routing and lifecycle handlers for the API.
package main

import "net/http"

// healthcheckHandler returns a JSON response indicating service availability, environment and version.
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	// Construct payload
	env := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.env,
			"version":     version,
		},
	}

	// Send 200 Ok with the serialized JSON envelope
	err := app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		// Log error and respond with 500 Internal Server Error.
		app.serverErrorResponse(w, r, err)
	}
}
