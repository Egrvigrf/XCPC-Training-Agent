package app

import (
	"errors"

	"aATA/internal/errno"
	"aATA/pkg/httpx"

	"github.com/gin-gonic/gin"
)

func InitErrorHandler() {
	httpx.SetErrorHandler(func(ctx *gin.Context, err error) (int, error) {
		switch err {

		case errno.ErrUserAlreadyExists:
			return 409, err
		case errno.ErrUserNotFound:
			return 404, err
		case errno.ErrPasswordInvalid,
			errno.ErrPasswordMismatch,
			errno.ErrPasswordEmpty,
			errno.ErrPasswordSame:
			return 400, err

		default:
			return 500, errors.New("internal server error")
		}
	})
}
