package internal

import "github.com/pkg/errors"

const (
	ExecutorIDCookieName        = "EXECUTOR_ID"
	ExecutorSignatureCookieName = "EXECUTOR_SIGNATURE"
)

var (
	ExecutorAlreadyExistsError = errors.New("executor already exists")
	ExecutorIDRequiredError    = errors.Errorf("%s is required", ExecutorIDCookieName)
	ExecutorNotFoundError      = errors.New("executor not found")
)
