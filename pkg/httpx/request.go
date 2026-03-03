package httpx

import "github.com/gin-gonic/gin"

func BindAndValidate(ctx *gin.Context, v any) error {
	// 绑定 Body 值
	if err := ctx.ShouldBind(v); err != nil {
		return err
	}

	// 绑定路径参数
	if err := ctx.ShouldBindUri(v); err != nil {
		return err
	}

	return nil
}
