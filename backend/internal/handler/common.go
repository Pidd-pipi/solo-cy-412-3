package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/dto"
	"net/http"
)

type Handler struct{ Validate *validator.Validate }

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, dto.Response{Code: constants.CodeOK, Message: constants.MessageOK, Data: data})
}
func Fail(c *gin.Context, status, code int, msg string) {
	c.JSON(status, dto.Response{Code: code, Message: msg})
}
func Bind(c *gin.Context, v any, validate *validator.Validate) bool {
	if e := c.ShouldBindJSON(v); e != nil || validate.Struct(v) != nil {
		Fail(c, 400, constants.CodeBadRequest, constants.MessageValidation)
		return false
	}
	return true
}
