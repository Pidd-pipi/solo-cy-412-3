package handler

import (
	"errors"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/dto"
	"github.com/smartestate/smartestate/internal/service"
)

type VisitorHandler struct {
	Handler
	svc *service.VisitorService
}

func NewVisitorHandler(s *service.VisitorService, h *Handler) *VisitorHandler {
	return &VisitorHandler{Handler: *h, svc: s}
}

// List 业主仅见本人凭证；物业/门岗可按 status、building 筛选。
func (h *VisitorHandler) List(c *gin.Context) {
	v, e := h.svc.List(c.GetUint("userID"), c.GetString("role"), c.Query("status"), c.Query("building"))
	if e != nil {
		failVisitor(c, e)
		return
	}
	OK(c, v)
}

// Create 业主登记访客凭证。
func (h *VisitorHandler) Create(c *gin.Context) {
	var r dto.CreateVisitorPassRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	start, e := service.ParseVisitTime(r.StartTime)
	if e != nil {
		Fail(c, 400, 40001, "到访开始时间格式不合法")
		return
	}
	end, e := service.ParseVisitTime(r.EndTime)
	if e != nil {
		Fail(c, 400, 40001, "到访结束时间格式不合法")
		return
	}
	v, err := h.svc.Create(c.GetUint("userID"), r.VisitorName, r.VisitorPhone, r.Building, r.Reason, start, end)
	if err != nil {
		failVisitor(c, err)
		return
	}
	OK(c, v)
}

// Detail 凭证详情及其生命周期留痕。
func (h *VisitorHandler) Detail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	v, events, e := h.svc.Detail(uint(id), c.GetUint("userID"), c.GetString("role"))
	if e != nil {
		failVisitor(c, e)
		return
	}
	OK(c, gin.H{"pass": v, "events": events})
}

// Cancel 业主取消本人凭证（物业亦可取消）。
func (h *VisitorHandler) Cancel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.Cancel(uint(id), c.GetUint("userID"), c.GetString("role"))
	if e != nil {
		failVisitor(c, e)
		return
	}
	OK(c, v)
}

// Approve 物业审核通过。
func (h *VisitorHandler) Approve(c *gin.Context) {
	var r dto.ReviewVisitorPassRequest
	if e := c.ShouldBindJSON(&r); e != nil && !errors.Is(e, io.EOF) {
		Fail(c, 400, 40001, constants.MessageValidation)
		return
	}
	r.Approved = true
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.Approve(uint(id), c.GetUint("userID"), r.Remark)
	if e != nil {
		failVisitor(c, e)
		return
	}
	OK(c, v)
}

// Reject 物业审核驳回。
func (h *VisitorHandler) Reject(c *gin.Context) {
	var r dto.ReviewVisitorPassRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.Reject(uint(id), c.GetUint("userID"), r.Remark)
	if e != nil {
		failVisitor(c, e)
		return
	}
	OK(c, v)
}
