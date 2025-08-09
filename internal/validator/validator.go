package validator

// Validator is a struct that holds validation errors. It provides a set of
// helper methods for performing validation checks and managing error messages.
type Validator struct {
	Errors map[string]string
}

//	Validator helpers

// Valid returns true if there are no validation errors.
func (v Validator) Valid() bool {
	return !v.HasErrors()
}

// HasErrors returns true if there are any validation errors.
func (v Validator) HasErrors() bool {
	return len(v.Errors) != 0
}

// AddError adds a new error message to the validator's error map. It will
// not overwrite an existing error.
func (v *Validator) AddError(key, message string) {
	if v.Errors == nil {
		v.Errors = map[string]string{}
	}

	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check adds an error message to the validator if a given condition is not met.
func (v *Validator) Check(key string, ok bool, message string) {
	if !ok {
		v.AddError(key, message)
	}
}
