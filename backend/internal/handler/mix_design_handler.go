package handler

import (
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/service"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type MixDesignHandler struct{ service service.MixDesignService }

func NewMixDesignHandler(value service.MixDesignService) *MixDesignHandler {
	return &MixDesignHandler{service: value}
}

func (handler *MixDesignHandler) List(c *gin.Context) {
	page, size := util.Pagination(c)
	result, err := handler.service.List(c.Request.Context(), dto.MixDesignQuery{
		Search: strings.TrimSpace(c.Query("search")), DesignState: c.Query("design_state"), Page: page, PageSize: size,
	})
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}

func (handler *MixDesignHandler) Get(c *gin.Context) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	result, err := handler.service.Get(c.Request.Context(), id)
	if err != nil {
		appError := util.NewError(http.StatusInternalServerError, util.CodeInternal, "unable to load mix design")
		util.WriteError(c, appError)
		return
	}
	util.OK(c, result)
}

func (handler *MixDesignHandler) Create(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.CreateMixDesignRequest
	if !util.BindJSON(c, &request) {
		return
	}
	result, err := handler.service.Create(c.Request.Context(), request, actor)
	if err != nil {
		appError := util.NewError(http.StatusInternalServerError, util.CodeInternal, "unable to create mix design")
		util.WriteError(c, appError)
		return
	}
	util.Created(c, result)
}

func (handler *MixDesignHandler) Update(c *gin.Context) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.UpdateMixDesignRequest
	if !util.BindJSON(c, &request) {
		return
	}
	result, err := handler.service.Update(c.Request.Context(), id, request, actor)
	if err != nil {
		appError := util.NewError(http.StatusInternalServerError, util.CodeInternal, "unable to update mix design")
		util.WriteError(c, appError)
		return
	}
	util.OK(c, result)
}

func (handler *MixDesignHandler) Validate(c *gin.Context) {
	handler.transition(c, constants.MixValidated)
}
func (handler *MixDesignHandler) Publish(c *gin.Context) {
	handler.transition(c, constants.MixPublished)
}
func (handler *MixDesignHandler) Retire(c *gin.Context) { handler.transition(c, constants.MixRetired) }

func (handler *MixDesignHandler) transition(c *gin.Context, state string) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.MixDesignActionRequest
	if !util.BindJSON(c, &request) {
		return
	}
	result, err := handler.service.Transition(c.Request.Context(), id, state, request, actor)
	if err != nil {
		appError := util.NewError(http.StatusInternalServerError, util.CodeInternal, "unable to transition mix design")
		util.WriteError(c, appError)
		return
	}
	util.OK(c, result)
}
