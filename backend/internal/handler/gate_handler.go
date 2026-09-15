package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/dto"
	"github.com/smartestate/smartestate/internal/service"
)

type GateHandler struct {
	Handler
	svc *service.VisitorService
}

func NewGateHandler(s *service.VisitorService, h *Handler) *GateHandler {
	return &GateHandler{Handler: *h, svc: s}
}

// Verify 门岗核对凭证（按凭证号），不改变状态，返回可否放行与原因。
func (h *GateHandler) Verify(c *gin.Context) {
	passNo := c.Query("pass_no")
	if passNo == "" {
		Fail(c, 400, 40001, "凭证号不能为空")
		return
	}
	v, info, e := h.svc.GateVerify(passNo, c.GetUint("userID"))
	if e != nil {
		failVisitor(c, e)
		return
	}
	OK(c, mapInfo(v, info))
}

// CheckIn 门岗办理进入。
func (h *GateHandler) CheckIn(c *gin.Context) {
	var r dto.GateCheckpointRequest
	if c.Request.ContentLength > 0 {
		if !Bind(c, &r, h.Validate) {
			return
		}
	}
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.CheckIn(uint(id), c.GetUint("userID"), r.Checkpoint)
	if e != nil {
		failVisitor(c, e)
		return
	}
	OK(c, v)
}

// CheckOut 门岗办理离开。
func (h *GateHandler) CheckOut(c *gin.Context) {
	var r dto.GateCheckpointRequest
	if c.Request.ContentLength > 0 {
		if !Bind(c, &r, h.Validate) {
			return
		}
	}
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.CheckOut(uint(id), c.GetUint("userID"), r.Checkpoint)
	if e != nil {
		failVisitor(c, e)
		return
	}
	OK(c, v)
}

// Records 门岗记录：最近的进出/审核留痕。
func (h *GateHandler) Records(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	v, e := h.svc.RecentEvents(limit)
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}

// mapInfo 组装门岗核对响应。
func mapInfo(v any, info map[string]any) map[string]any {
	info["pass"] = v
	return info
}
