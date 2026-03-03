package errno

import "errors"

// 用户相关业务错误
var (
	ErrUserAlreadyExists = errors.New("用户已存在")
	ErrUserNotFound      = errors.New("用户不存在")
	ErrPasswordInvalid   = errors.New("密码错误")

	ErrPasswordMismatch = errors.New("两次输入密码不一致")
	ErrPasswordEmpty    = errors.New("新密码不能为空")
	ErrPasswordSame     = errors.New("新密码不能与旧密码相同")
)
