package dbstorage

import "errors"

var ErrUserAlreadyExists = errors.New("user with such credentials already exist")
var ErrInvalidLoginPassword = errors.New("invalid login/password")
