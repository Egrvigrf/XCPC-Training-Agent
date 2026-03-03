package middleware

import "github.com/gin-gonic/gin"

type AdminMid struct{}

func NewAdminMid() *AdminMid {
	return &AdminMid{}
}

func (m *AdminMid) Handler(ctx *gin.Context) {
	v, ok := ctx.Get("is_admin")
	if !ok {
		ctx.AbortWithStatusJSON(403, gin.H{"msg": "无权限"})
		return
	}

	isAdmin, ok := v.(bool)
	if !ok || !isAdmin {
		ctx.AbortWithStatusJSON(403, gin.H{"msg": "无权限"})
		return
	}

	ctx.Next()
}
