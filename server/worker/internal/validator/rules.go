package validator

import (
	"regexp"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var userRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*(?:[._-][a-zA-Z0-9]+)*$`)

func registerCustomValidation(v *validator.Validate) {
	v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		return userRegex.MatchString(fl.Field().String())
	})
	v.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()

		if len(s) < 8 {
			return false
		}
		var hasUpper, hasLower, hasDigit, hasSpecial bool
		for _, r := range s {
			switch {
			case unicode.IsUpper(r):
				hasUpper = true
			case unicode.IsLower(r):
				hasLower = true
			case unicode.IsDigit(r):
				hasDigit = true
			case unicode.IsSymbol(r), unicode.IsPunct(r):
				hasSpecial = true
			}
		}
		return hasUpper && hasLower && hasDigit && hasSpecial
	})
}
