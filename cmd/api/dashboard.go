// Package main contains the HTTP server, application handlers, and supporting
// functionality for the gotp application/
package main

import (
	"net/http"

	"github.com/codercollo/gOTP/web"
)

// dashboardHandler serves the embedded single=page application dashboard.
func (app *application) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Read the dashboard HTML from the embedded filesystem.
	data, err := web.Files.ReadFile("static/index.html")
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, err = w.Write(data)
	if err != nil {
		app.logger.PrintError(err, nil)
	}
}
