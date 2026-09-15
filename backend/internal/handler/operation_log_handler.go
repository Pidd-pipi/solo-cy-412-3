package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/service"
)

type OperationLogHandler struct{ svc *service.OperationLogService }

func NewOperationLogHandler(s *service.OperationLogService) *OperationLogHandler {
	return &OperationLogHandler{s}
}
func (h *OperationLogHandler) List(c *gin.Context) {
	v, e := h.svc.List()
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}
