package domain

import "errors"

var (
	ErrNotFound     = errors.New("asset not found")
	ErrTooLarge     = errors.New("object exceeds size policy")
	ErrEmpty        = errors.New("object is empty")
	ErrMIME         = errors.New("mime type is not routable")
	ErrTenant       = errors.New("tenant id is invalid")
	ErrHash         = errors.New("content hash is invalid")
	ErrHashMismatch = errors.New("content hash mismatch")
	ErrNoNodes      = errors.New("no healthy edge nodes")
)
