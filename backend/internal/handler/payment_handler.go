package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/dto"
	"github.com/smartestate/smartestate/internal/service"
	"strconv"
)

type PaymentHandler struct {
	Handler
	svc *service.PaymentService
}

func NewPaymentHandler(s *service.PaymentService, h *Handler) *PaymentHandler {
	return &PaymentHandler{Handler: *h, svc: s}
}
func (h *PaymentHandler) List(c *gin.Context) {
	uid := c.GetUint("userID")
	if c.GetString("role") != "resident" {
		uid = 0
	}
	v, e := h.svc.List(uid)
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}
func (h *PaymentHandler) Create(c *gin.Context) {
	var r dto.CreatePaymentRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	v, e := h.svc.Create(r.UserID, r.FeeType, r.Amount, r.Month)
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}
func (h *PaymentHandler) Pay(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.Pay(uint(id), c.GetUint("userID"), c.GetString("role"))
	if e != nil {
		Fail(c, 400, 40001, e.Error())
		return
	}
	OK(c, v)
}
