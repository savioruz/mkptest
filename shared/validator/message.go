package validator

import (
	"errors"
	"strings"

	val "github.com/go-playground/validator/v10"
)

var (
	messages = map[string]string{
		"required": "{field} is required",
		"gte":      "{field} must be greater than or equal to {param}",
		"lte":      "{field} must be less than or equal to {param}",
		"oneof":    "{field} must be one of {param}",
		"max":      "{field} must be less than or equal to {param}",
		"min":      "{field} must be greater than or equal to {param}",
		"email":    "{field} must be a valid email address",
		"url":      "{field} must be a valid URL",
		"dive":     "{field} contains a forbidden value",
		"mimetype": "{field} must be one of the allowed file types: {param}",
	}
)

func message(err error) string {
	var valErrors val.ValidationErrors

	if errors.As(err, &valErrors) {
		for _, valErr := range valErrors {
			errStr := ""
			field := valErr.Field()
			param := valErr.Param()

			errStr = messages[valErr.Tag()]
			if errStr != "" {
				errStr = strings.ReplaceAll(errStr, "{field}", field)
				errStr = strings.ReplaceAll(errStr, "{param}", param)

				return errStr
			}
		}

		return valErrors.Error()
	}

	return err.Error()
}
