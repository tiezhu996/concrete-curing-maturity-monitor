package handler

import (
	"concrete-curing-maturity-monitor/backend/internal/middleware"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct{ auth *middleware.Authenticator }

func NewAuthHandler(auth *middleware.Authenticator) *AuthHandler { return &AuthHandler{auth: auth} }

type loginRequest struct {
	Username string `json:"username" binding:"required,min=2,max=80"`
	Password string `json:"password" binding:"required,min=6,max=120"`
}

func (handler *AuthHandler) Login(c *gin.Context) {
	var request loginRequest
	if !util.BindJSON(c, &request) {
		return
	}
	session, err := handler.auth.Login(c.Request.Context(), strings.TrimSpace(request.Username), request.Password)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, session)
}

func requireActor(c *gin.Context) (util.Actor, bool) {
	actor, ok := util.ActorFromGin(c)
	if !ok {
		util.WriteError(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "authentication is required"))
		return util.Actor{}, false
	}
	return actor, true
}
