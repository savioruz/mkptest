package validator

import (
	"encoding/json"
	"fmt"
	val "github.com/go-playground/validator/v10"
	"io"
	"mime/multipart"
	"oil/config"
	"oil/shared/base64"
	"oil/shared/constant"
	"oil/shared/failure"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

var validate *val.Validate

func registerMimetypeValidation(field val.FieldLevel) bool {
	var contentType string

	if file, ok := field.Field().Interface().(multipart.FileHeader); ok {
		contentType = file.Header.Get(constant.RequestHeaderContentType)
	} else if str, ok := field.Field().Interface().(string); ok {
		contentType = base64.GetContentType(str)

		if contentType == "" {
			return false
		}
	}

	allowedTypes := strings.Split(field.Param(), " ")

	return slices.Contains(allowedTypes, contentType)
}

func registerFileSizeValidation(field val.FieldLevel) bool {
	fileSize := 0
	if file, ok := field.Field().Interface().(multipart.FileHeader); ok {
		fileSize = int(file.Size)
	} else if str, ok := field.Field().Interface().(string); ok {
		fileSize = len(str)
	}

	maxSizeMB, err := strconv.ParseFloat(field.Param(), 64)
	if err != nil {
		return false
	}

	bytesConversion := 1024.0
	maxSizeBytes := int(maxSizeMB * bytesConversion * bytesConversion)

	return fileSize <= maxSizeBytes
}

func init() {
	cfg := config.Get()

	validate = val.New(val.WithRequiredStructEnabled())
	err := validate.RegisterValidation("oil", func(fl val.FieldLevel) bool {
		method := fl.Field().MethodByName("Validate")
		if method.IsValid() {
			result := method.Call([]reflect.Value{reflect.ValueOf(cfg)})

			return result[0].Interface() == nil
		}

		return false
	})

	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("empty", func(fl val.FieldLevel) bool {
		empty := fl.Field().IsZero()

		return empty
	})

	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("mimetypes", registerMimetypeValidation)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("maxfilesize", registerFileSizeValidation)
	if err != nil {
		panic(err)
	}
}

// Validate reads from the given io.Reader into the given struct, and then performs validation
// on the struct using the validator package. If the struct is invalid according to the
// validation rules, an error is returned. Otherwise, nil is returned.
// https://github.com/go-playground/validator
func Validate[T any](r io.Reader, data *T) error {
	decoder := json.NewDecoder(r)
	err := decoder.Decode(data)

	if err != nil {
		return failure.BadRequest(fmt.Errorf("failed to decode request body: %w", err)) //nolint:wrapcheck
	}

	return ValidateStruct(data)
}

func ValidateStruct[T any](data *T) error {
	err := validate.Struct(data)

	if err != nil {
		msg := message(err)

		return failure.BadRequestFromString(msg) //nolint:wrapcheck
	}

	return nil
}

func ValidateVar(field any, tag string) error {
	err := validate.Var(field, tag)

	if err != nil {
		msg := message(err)

		return failure.BadRequestFromString(msg) //nolint:wrapcheck
	}

	return nil
}
