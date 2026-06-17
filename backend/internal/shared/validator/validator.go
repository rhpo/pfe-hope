package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

func Struct(s any) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	var msgs []string
	for _, e := range validationErrors {
		msgs = append(msgs, translateError(e))
	}

	return fmt.Errorf("%s", strings.Join(msgs, "; "))
}

func translateError(e validator.FieldError) string {
	field := e.Field()
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("Le champ '%s' est obligatoire", field)
	case "email":
		return fmt.Sprintf("Le champ '%s' doit être une adresse email valide", field)
	case "min":
		return fmt.Sprintf("Le champ '%s' doit avoir au moins %s caractères", field, e.Param())
	case "max":
		return fmt.Sprintf("Le champ '%s' doit avoir au maximum %s caractères", field, e.Param())
	case "len":
		return fmt.Sprintf("Le champ '%s' doit avoir exactement %s caractères", field, e.Param())
	case "oneof":
		return fmt.Sprintf("Le champ '%s' doit être l'une des valeurs suivantes: %s", field, e.Param())
	case "url":
		return fmt.Sprintf("Le champ '%s' doit être une URL valide", field)
	default:
		return fmt.Sprintf("Le champ '%s' est invalide (%s)", field, e.Tag())
	}
}

func Var(field any, tag string) error {
	return validate.Var(field, tag)
}
