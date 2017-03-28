package errmsg

import (
	"errors"
)

var (
	// ErrRecordNotFound record not found error, happens when haven't find any matched data when looking up with a struct
	ErrRecordNotFound = errors.New("ngorm: record not found")

	// ErrInvalidSQL invalid SQL error, happens when you passed invalid SQL
	ErrInvalidSQL = errors.New("ngorm: invalid SQL")

	// ErrInvalidTransaction invalid transaction when you are trying to `Commit` or `Rollback`
	ErrInvalidTransaction = errors.New("ngorm: no valid transaction")

	// ErrCantStartTransaction can't start transaction when you are trying to start one with `Begin`
	ErrCantStartTransaction = errors.New("ngorm: can't start transaction")

	// ErrUnaddressable unaddressable value
	ErrUnaddressable = errors.New("ngorm: using unaddressable value")

	//ErrInvalidFieldValue invalid field value
	ErrInvalidFieldValue = errors.New("ngorm: field value not valid")

	// ErrMissingModel when the struct model is not set for the database operation
	ErrMissingModel = errors.New("missing model")
)
