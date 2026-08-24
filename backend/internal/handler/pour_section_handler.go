package handler

import (
	"context"
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/service"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type PourSectionHandler struct{ service service.PourSectionService }

func NewPourSectionHandler(value service.PourSectionService) *PourSectionHandler {
	return &PourSectionHandler{service: value}
}

func (handler *PourSectionHandler) List(c *gin.Context) {
	page, size := util.Pagination(c)
	mixID, _ := strconv.ParseUint(c.Query("mix_design_id"), 10, 64)
	result, err := handler.service.List(context.Background(), dto.PourSectionQuery{
		Search: strings.TrimSpace(c.Query("search")), CuringState: c.Query("curing_state"),
		MixDesignID: uint(mixID), OwnerTeam: c.Query("owner_team"), Page: page, PageSize: size,
	})
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}

func (handler *PourSectionHandler) Get(c *gin.Context) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	result, err := handler.service.Get(context.Background(), id)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}

func (handler *PourSectionHandler) Create(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.CreatePourSectionRequest
	if !util.BindJSON(c, &request) {
		return
	}
	result, err := handler.service.Create(c.Request.Context(), request, actor)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.Created(c, result)
}

func (handler *PourSectionHandler) Update(c *gin.Context) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.UpdatePourSectionRequest
	if !util.BindJSON(c, &request) {
		return
	}
	result, err := handler.service.Update(c.Request.Context(), id, request, actor)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}

func (handler *PourSectionHandler) Transition(c *gin.Context) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.TransitionPourSectionRequest
	if !util.BindJSON(c, &request) {
		return
	}
	result, err := handler.service.Transition(c.Request.Context(), id, request, actor)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}
