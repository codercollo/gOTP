// Package main configures HTTP routing and endpoint registration.
package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// routes builds and returns the http.Handler with registered endpoints and error handlers.
func (app *application) routes() http.Handler {
	router := httprouter.New()

	// Override default error handlers with custom JSON responses
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// Application routes
	router.HandlerFunc(http.MethodGet, "/v1/heatlthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/otp/send", app.sendOTPHandler)
	router.HandlerFunc(http.MethodPost, "/v1/otp/verify", app.verifyOTPHandler)

	router.HandlerFunc(http.MethodGet, "/", app.dashboardHandler)

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return app.metrics(app.recoverPanic(app.secureHeaders(app.rateLimit(app.authenticate(app.requestLog(router))))))

}
