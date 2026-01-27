package repository 

import "errors"

var ErrNotFound = errors.New("задача не найдена")
var ErrVersionConflict = errors.New("конфликт версий")