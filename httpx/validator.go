package httpx

import "github.com/go-playground/validator/v10"

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	return &Validator{
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (v *Validator) Validate(target any) error {
	err := v.validate.Struct(target)
	return err
}
