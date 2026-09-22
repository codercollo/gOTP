// Package main handlers OTP generation, dispatch and verification requests.
package main

import (
	"net/http"

	"github.com/codercollo/gOTP/internal/data"
	"github.com/codercollo/gOTP/internal/otp"
	"github.com/codercollo/gOTP/internal/validator"
)

// sendOTPHandler generates an OTP code, save its hash and dispatches it asynchronously.
func (app *application) sendOTPHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var input struct {
		PhoneNumber string `json:"phone_number"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate phone number format.
	v := validator.New()
	v.Check(input.PhoneNumber != "", "phone_number", "must be provided")
	v.Check(validator.Matches(input.PhoneNumber, validator.PhoneRX), "phone_number", "must be a valid E.164 number, e.g. +254712345678")
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Generate numeric OTP code.
	code, err := otp.GenerateNumeric(app.config.otp.length)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Save hashed	OTP to data store.
	app.models.OTP.Insert(input.PhoneNumber, otp.Hash(code), app.config.otp.ttl, app.config.otp.maxAttempts)

	// Dispatch SMS in background worker.
	app.background(func() {
		if app.config.env == "development" {
			app.logger.PrintInfo("otp generated", map[string]string{
				"phone": input.PhoneNumber,
				"code":  code,
			})
		} else {
			app.logger.PrintInfo("otp dispatched", map[string]string{
				"phone": input.PhoneNumber,
			})
		}
	})

	// Respond with 202 Accepted
	env := envelope{"message": "if the number is valid, a code has been sent"}
	if err := app.writeJSON(w, http.StatusAccepted, env, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// verifyOTPHandler validates a submitted OTP code and consumes it atomically.
func (app *application) verifyOTPHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var input struct {
		PhoneNumber string `json:"phone_number"`
		Code        string `json:"code"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate required fields
	v := validator.New()
	v.Check(input.PhoneNumber != "", "phone_number", "must be provided")
	v.Check(validator.Matches(input.PhoneNumber, validator.PhoneRX), "phone_number", "must be a valid E.164 number")
	v.Check(input.Code != "", "code", "must be provided")
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Verify and consume code atomically.
	result := app.models.OTP.VerifyAndConsume(input.PhoneNumber, input.Code)

	// Map verification results to HTTP response
	switch result {
	case data.ResultOK:
		env := envelope{"status": "verified"}
		if err := app.writeJSON(w, http.StatusOK, env, nil); err != nil {
			app.serverErrorResponse(w, r, err)
		}
	case data.ResultLocked:
		app.errorResponse(w, r, http.StatusTooManyRequests, "too many attempts; request a new code")
	case data.ResultNoCode, data.ResultExpired, data.ResultMismatch:
		// Generic 422 error to avoid leaking user/code status
		app.errorResponse(w, r, http.StatusUnprocessableEntity, "invalid or expired code")
	default:
		app.errorResponse(w, r, http.StatusInternalServerError, "the server encountered a problem and could not process your request")
	}
}
