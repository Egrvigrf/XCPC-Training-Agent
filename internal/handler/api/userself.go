package api

import (
	"github.com/gin-gonic/gin"

	"aATA/internal/domain"
	"aATA/internal/logic"
	"aATA/internal/svc"
	"aATA/pkg/httpx"
)

type UserSelf struct {
	svcCtx *svc.ServiceContext
	user   logic.User
}

func NewUserSelf(svcCtx *svc.ServiceContext, user logic.User) *UserSelf {
	return &UserSelf{
		svcCtx: svcCtx,
		user:   user,
	}
}

func (h *UserSelf) InitRegister(engine *gin.Engine) {
	g := engine.Group("v1/user", h.svcCtx.JwtMid.Handler)
	g.GET("/me", h.Info)
	g.POST("/password", h.UpPassword)
	g.DELETE("/me", h.DeleteSelf)
}

func (h *UserSelf) Info(ctx *gin.Context) {
	uid, err := h.svcCtx.JWT.GetUID(ctx)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	res, err := h.user.Info(ctx.Request.Context(), int64(uid))
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, res)
}

func (h *UserSelf) UpPassword(ctx *gin.Context) {
	var req domain.UpPasswordReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}
	uid, err := h.svcCtx.JWT.GetUID(ctx)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	err = h.user.UpPassword(ctx.Request.Context(), uid, &req)
	if err != nil {
		httpx.FailWithErr(ctx, err)
	} else {
		httpx.Ok(ctx)
	}
}

func (h *UserSelf) DeleteSelf(ctx *gin.Context) {
	uid, err := h.svcCtx.JWT.GetUID(ctx)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}
	err = h.user.DeleteSelf(ctx.Request.Context(), uid)
	if err != nil {
		httpx.FailWithErr(ctx, err)
	} else {
		httpx.Ok(ctx)
	}
}
