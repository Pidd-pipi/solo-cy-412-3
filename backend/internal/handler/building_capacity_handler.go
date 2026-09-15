package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/dto"
	"github.com/smartestate/smartestate/internal/service"
)

type BuildingCapacityHandler struct {
	Handler
	svc *service.VisitorService
}

func NewBuildingCapacityHandler(s *service.VisitorService, h *Handler) *BuildingCapacityHandler {
	return &BuildingCapacityHandler{Handler: *h, svc: s}
}

// Overview 各楼栋当日容量上限 / 在场 / 剩余。
func (h *BuildingCapacityHandler) Overview(c *gin.Context) {
	v, e := h.svc.CapacityOverview()
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}

// Update 物业调整某楼栋当日容量上限。
func (h *BuildingCapacityHandler) Update(c *gin.Context) {
	var r dto.BuildingCapacityRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	v, e := h.svc.SetCapacity(r.Building, r.DailyLimit, c.GetUint("userID"))
	if e != nil {
		failVisitor(c, e)
		return
	}
	OK(c, v)
}
