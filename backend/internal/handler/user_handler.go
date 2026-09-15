package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/dto"
	"github.com/smartestate/smartestate/internal/service"
)

type UserHandler struct {
	Handler
	svc *service.UserService
}

func NewUserHandler(s *service.UserService, h *Handler) *UserHandler {
	return &UserHandler{Handler: *h, svc: s}
}
func (h *UserHandler) Me(c *gin.Context) {
	v, e := h.svc.Me(c.GetUint("userID"))
	if e != nil {
		Fail(c, 404, 40401, "用户不存在")
		return
	}
	OK(c, v)
}
func (h *UserHandler) UpdateMe(c *gin.Context) {
	var r dto.UpdateProfileRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	v, e := h.svc.Update(c.GetUint("userID"), r.Nickname, r.Avatar, r.Building, r.Unit, r.Room)
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}
func (h *UserHandler) Staff(c *gin.Context) {
	v, e := h.svc.Staff()
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}
