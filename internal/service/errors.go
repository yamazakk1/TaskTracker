package service

import "fmt"

type BusinessError struct{
	Code string
	Message string
	Details map[string]any
	Err error 
}

type Detail struct{
	Key string 
	Paylod any
}

func (b *BusinessError) Error() string{
	if b.Err != nil{
		return fmt.Sprintf("[%s] %s: %s", b.Code, b.Message, b.Err.Error())
	}
	return fmt.Sprintf("[%s] %s", b.Code, b.Message)
}

func ToDetail(key string, payload any) Detail{
	return Detail{
		Key: key,
		Paylod: payload,
	}
}

func NewBusinessError(code string, message string, details ...Detail) *BusinessError{
	BusErr := &BusinessError{
		Code: code,
		Message: message,
		Details: make(map[string]any),
	}

	for _,detail := range details{
		BusErr.Details[detail.Key] = detail.Paylod
	}

	return BusErr
}

func NewNotFound(resource RepoType, id string) *BusinessError {
    return &BusinessError{
        Code:    "NOT_FOUND",
        Message: fmt.Sprintf("%s %s не найден(а)", resource, id),
        Details: map[string]any{
            "resource": resource,
            "id":       id,
        },
    }
}

func NewValidationError(field, reason string) *BusinessError {
    return &BusinessError{
        Code:    "VALIDATION_ERROR",
        Message: fmt.Sprintf("Неверное значение поля '%s': %s", field, reason),
        Details: map[string]any{
            "field":  field,
            "reason": reason,
        },
    }
}