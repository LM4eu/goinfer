package errors

import "errors"

// Standard application errors
var (
    ErrModelNotFound      = errors.New("model not found")
    ErrInvalidInput       = errors.New("invalid input")
    ErrModelLoadFailed    = errors.New("failed to load model")
    ErrTemplateParseError = errors.New("template parsing error")
)
