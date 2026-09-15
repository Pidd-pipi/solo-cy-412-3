package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/service"
)

// failVisitor 将服务层哨兵错误映射为合适的 HTTP 状态与统一错误码。
func failVisitor(c *gin.Context, e error) {
	switch {
	case errors.Is(e, service.ErrPassNotFound):
		Fail(c, http.StatusNotFound, constants.CodeNotFound, e.Error())
	case errors.Is(e, service.ErrPassForbidden):
		Fail(c, http.StatusForbidden, constants.CodeForbidden, e.Error())
	case errors.Is(e, service.ErrPassOverlap), errors.Is(e, service.ErrCapacityReached):
		Fail(c, http.StatusConflict, constants.CodeConflict, e.Error())
	case errors.Is(e, service.ErrPassState), errors.Is(e, service.ErrPassTimeInvalid),
		errors.Is(e, service.ErrGateNotInWindow), errors.Is(e, service.ErrGateAlreadyIn),
		errors.Is(e, service.ErrGateNotIn):
		Fail(c, http.StatusBadRequest, constants.CodeBadRequest, e.Error())
	default:
		Fail(c, http.StatusInternalServerError, constants.CodeInternal, e.Error())
	}
}
