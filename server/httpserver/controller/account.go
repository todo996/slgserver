package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/llr104/slgserver/constant"
	"github.com/llr104/slgserver/log"
	myhttp "github.com/llr104/slgserver/server/httpserver"
	"github.com/llr104/slgserver/server/httpserver/logic"
)

type AccountController struct{}

func (self AccountController) RegisterRoutes(group *echo.Group) {
	group.POST("/account/register", self.register)
	group.POST("/account/changepwd", self.changePassword)
	group.POST("/account/forgetpwd", self.notImplemented)
	group.POST("/account/resetpwd", self.notImplemented)
}

func (self AccountController) register(ctx echo.Context) error {
	log.DefaultLog.Info("Đăng ký tài khoản")
	if err := logic.DefaultUser.CreateUser(ctx); err != nil {
		return writeAccountError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": constant.OK,
	})
}

func (self AccountController) changePassword(ctx echo.Context) error {
	log.DefaultLog.Info("Đổi mật khẩu")
	if err := logic.DefaultUser.ChangePassword(ctx); err != nil {
		return writeAccountError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"code": constant.OK,
	})
}

func (self AccountController) notImplemented(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, map[string]interface{}{
		"code":   constant.InvalidParam,
		"errmsg": "Chức năng này chưa được hỗ trợ.",
	})
}

func writeAccountError(ctx echo.Context, err error) error {
	apiError := &myhttp.MyError{}
	if errors.As(err, &apiError) {
		return ctx.JSON(http.StatusOK, map[string]interface{}{
			"code":   apiError.Id(),
			"errmsg": apiError.Error(),
		})
	}

	log.DefaultLog.Error("Lỗi API tài khoản")
	return ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
		"code":   constant.DBError,
		"errmsg": "Máy chủ gặp lỗi. Vui lòng thử lại sau.",
	})
}
