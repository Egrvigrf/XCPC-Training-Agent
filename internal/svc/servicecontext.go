package svc

import (
	"aATA/internal/config"
	"aATA/internal/middleware"
	"aATA/internal/model"
	"aATA/pkg/encrypt"
	"aATA/pkg/jwt"
	"context"
	"errors"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config config.Config

	// 基础设施
	JWT        *jwt.JWT
	UsersModel model.UsersModel

	// Middleware
	JwtMid     *middleware.JWTMid
	AdminMid   *middleware.AdminMid
	LoggingMid *middleware.LoggingMid
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	db, err := gorm.Open(mysql.Open(c.MySql.DataSource), &gorm.Config{}) // 这就是进行组装了
	if err != nil {
		return nil, err
	}

	jwtTool := jwt.NewJWT(
		c.JWT.Secret,
		c.JWT.Expire,
	)

	res := &ServiceContext{
		Config:     c,
		UsersModel: model.NewUsersModel(db),
		JWT:        jwtTool,
		JwtMid:     middleware.NewJWTMid(jwtTool),
		LoggingMid: middleware.NewLoggingMid(),
	}

	return res, initServer(res)
}

func initServer(svc *ServiceContext) error {
	ctx := context.Background()
	if err := initSystemUser(ctx, svc); err != nil {
		return err
	}
	return nil
}

func initSystemUser(ctx context.Context, svc *ServiceContext) error {
	systemUser, err := svc.UsersModel.SystemUser()

	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}

	if systemUser != nil {
		return nil
	}

	// 防止重复 root
	u, err := svc.UsersModel.FindByName("root")
	if err == nil && u != nil {
		return nil
	}

	pwd, err := encrypt.GenPasswordHash([]byte("000000"))
	if err != nil {
		return err
	}

	return svc.UsersModel.Insert(ctx, &model.Users{
		Name:     "root",
		Phone:    "123456789",
		Password: string(pwd),
		Status:   model.UserStatusNormal,
		IsSystem: model.IsSystemUser,
	})
}
