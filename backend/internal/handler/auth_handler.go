package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/dto"
	"github.com/smartestate/smartestate/internal/service"
	"github.com/smartestate/smartestate/internal/util"
	"net/http"
)

type AuthHandler struct {
	Handler
	users  *service.UserService
	secret string
}

func NewAuthHandler(u *service.UserService, secret string, v *Handler) *AuthHandler {
	return &AuthHandler{Handler: *v, users: u, secret: secret}
}
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !Bind(c, &req, h.Validate) {
		return
	}
	u, e := h.users.Login(req.Phone, req.Password)
	if e != nil {
		Fail(c, http.StatusUnauthorized, constants.CodeUnauthorized, constants.MessageUnauthorized)
		return
	}
	token, e := util.SignJWT(h.secret, u.ID, u.Role)
	if e != nil {
		Fail(c, 500, constants.CodeInternal, "token generation failed")
		return
	}
	OK(c, gin.H{"token": token, "user": u})
}
