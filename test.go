package main

import (
	"testing"
)

// test fields validation function
func TestFieldsValidation(t *testing.T) {

	FirstName := "maksin"
	LastName := "guberniev"
	Gender := "null"
	Address := "undefined"
	Email := "email.com"

	result := validateForm(&FirstName, &LastName, &Gender, &Address, &Email)

	if !result {
		t.Error("Error occured while validating.", result)
	}

}