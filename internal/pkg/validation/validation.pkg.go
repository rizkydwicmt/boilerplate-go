package validation

import (
	"boilerplate-go/internal/common/enum"
	types "boilerplate-go/internal/common/type"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"

	"github.com/go-playground/validator/v10"
)

var val *validator.Validate

var validationMessages = map[string]string{
	"e164":         "must be a e164 formatted phone number",
	"required":     "is required",
	"url":          "must be a valid URL",
	"datetime":     "must be a valid date-time format (2006-01-02T15:04:05Z07:00)",
	"number":       "must be a number",
	"oneof":        "must be one of the allowed values: %s",
	"email":        "must be a valid email address",
	"min":          "must be greater than or equal to %s",
	"max":          "must be less than or equal to %s",
	"len":          "must have the exact length of %s",
	"alpha":        "must contain only alphabetic characters",
	"alphanum":     "must contain only alphanumeric characters",
	"eqfield":      "must be equal to the value of the %s field",
	"nefield":      "must not be equal to the value of the %s field",
	"gt":           "must be greater than %s",
	"gte":          "must be greater than or equal to %s",
	"lt":           "must be less than %s",
	"lte":          "must be less than or equal to %s",
	"excludes":     "must not contain the value %s",
	"excludesall":  "must not contain any of the values: %s",
	"enum":         "must be one of the allowed enum values: %s",
	"stringToBool": "must be a boolean value",
}

func Setup() error {
	val = validator.New(validator.WithRequiredStructEnabled())

	if err := RegisterValidations(val); err != nil {
		return fmt.Errorf("failed to register custom validations: %w", err)
	}

	val.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := RegisterValidations(v); err != nil {
			return fmt.Errorf("failed to register custom validations in Gin engine: %w", err)
		}
	} else {
		return fmt.Errorf("failed to get validation engine")
	}

	return nil
}

func RegisterValidations(v *validator.Validate) error {
	if err := v.RegisterValidation("enum", enum.ValidateEnum); err != nil {
		return fmt.Errorf("failed to register enum validation: %w", err)
	}
	if err := v.RegisterValidation("stringToBool", types.ValidateStringToBool); err != nil {
		return fmt.Errorf("failed to register stringToBool validation: %w", err)
	}
	return nil
}

func Validate(payload interface{}) error {
	if err := val.Struct(payload); err != nil {
		var errorMessages []string

		validationErrors := parsingErrorValidate(err)
		if validationErrors != "" {
			errorMessages = append(errorMessages, validationErrors)
		}
		message := "Validation failed: " + strings.Join(errorMessages, ", ")
		return errors.New(message)
	}

	return nil
}

func parsingErrorValidate(err error) string {
	var errs validator.ValidationErrors
	if errors.As(err, &errs) {
		var sb strings.Builder
		for _, e := range errs {
			name := e.Namespace()
			field := e.Field()
			tag := e.Tag()
			param := e.Param()
			tp := e.Type()

			msg := validationMessages[tag]
			switch tag {
			case "enum":
				msg = fmt.Sprintf(msg, tp)
			default:
				if strings.Contains(msg, "%s") {
					msg = fmt.Sprintf(msg, param)
				}
			}
			sb.WriteString(fmt.Sprintf("%s: %s %s", name, field, msg))
			sb.WriteString(", ")
		}
		return strings.TrimSuffix(sb.String(), ", ")
	}
	return err.Error()
}
