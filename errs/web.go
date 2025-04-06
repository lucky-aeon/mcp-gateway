package errs

import "errors"

// 鉴权失败
var (
	ErrAuthFailed         = errors.New("auth_failed, invalid api key")
	ErrAuthConfigNotFound = errors.New("auth_config_not_found")
)
