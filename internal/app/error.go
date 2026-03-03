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

		// ===== User =====
		case errno.ErrUserAlreadyExists:
			return 409, err
		case errno.ErrUserNotFound:
			return 404, err
		case errno.ErrPasswordInvalid,
			errno.ErrPasswordMismatch,
			errno.ErrPasswordEmpty,
			errno.ErrPasswordSame:
			return 400, err

		// ===== File =====
		case errno.ErrInvalidPath:
			return 400, err
		case errno.ErrAlreadyExist:
			return 409, err
		case errno.ErrDirNotEmpty:
			return 400, err
		case errno.ErrNotExist:
			return 404, err

		default:
			return 500, errors.New("internal server error")
		}
	})
}
