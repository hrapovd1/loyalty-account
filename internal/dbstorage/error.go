package dbstorage

import "errors"

var ErrUserAlreadyExists = errors.New("user with such credentials already exist")
var ErrInvalidLoginPassword = errors.New("invalid login/password")
var ErrOrderExists = errors.New("order early uploaded")
var ErrOrderExistsAnother = errors.New("order early uploaded another user")
var ErrNoOrders = errors.New("orders not found")
var ErrNotEnoughFunds = errors.New("not enough funds")
