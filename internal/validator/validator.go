// Package validator provides input validation helpers and error map collection.
package validator

import "regexp"

// PhoneRX matches E.164 phone numbers (+ followed by 7 to 15 digits).
var PhoneRX = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

// Validator holds field-level validation error messages mapped by key.
type Validator struct {
	Errors map[string]string // Validation error map
}

// New initializes and returns a new Validator instance.
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// Valid returns true if no validation errors have been recorded.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError records an error message for key if one does not already exist.
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check adds an error message to the map if ok evaluates to false.
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// Matches reports whether value matches the regular expression pattern.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}
