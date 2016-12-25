package errmsg

import "errors"

var (
	// ErrRecordNotFound record not found error, happens when haven't find any matched data when looking up with a struct
	RecordNotFound = errors.New("record not found")
	// ErrInvalidSQL invalid SQL error, happens when you passed invalid SQL
	InvalidSQL = errors.New("invalid SQL")
	// ErrInvalidTransaction invalid transaction when you are trying to `Commit` or `Rollback`
	InvalidTransaction = errors.New("no valid transaction")
	// ErrCantStartTransaction can't start transaction when you are trying to start one with `Begin`
	CantStartTransaction = errors.New("can't start transaction")
	// ErrUnaddressable unaddressable value
	Unaddressable = errors.New("using unaddressable value")
)
