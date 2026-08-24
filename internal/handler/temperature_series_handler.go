package handler

import (
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/service"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type TemperatureSeriesHandler struct {
	service service.TemperatureSeriesService
}

func NewTemperatureSeriesHandler(value service.TemperatureSeriesService) *TemperatureSeriesHandler {
	return &TemperatureSeriesHandler{service: value}
}

func (handler *TemperatureSeriesHandler) List(c *gin.Context) {
	page, size := util.Pagination(c)
	sectionID, _ := strconv.ParseUint(c.Query("pour_section_id"), 10, 64)
	result, err := handler.service.List(c.Request.Context(), dto.TemperatureSeriesQuery{
		PourSectionID: uint(sectionID), SeriesState: c.Query("series_state"),
		SensorCode: strings.TrimSpace(c.Query("sensor_code")), Page: page, PageSize: size,
	})
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}

func (handler *TemperatureSeriesHandler) Get(c *gin.Context) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	result, err := handler.service.Get(c.Request.Context(), id)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}

func (handler *TemperatureSeriesHandler) Import(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.ImportTemperatureSeriesRequest
	if !util.BindJSON(c, &request) {
		return
	}
	result, err := handler.service.Import(c.Request.Context(), request, actor)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.Created(c, result)
}

func (handler *TemperatureSeriesHandler) Confirm(c *gin.Context)    { handler.action(c, true) }
func (handler *TemperatureSeriesHandler) Invalidate(c *gin.Context) { handler.action(c, false) }

func (handler *TemperatureSeriesHandler) action(c *gin.Context, confirm bool) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.TemperatureSeriesActionRequest
	if !util.BindJSON(c, &request) {
		return
	}
	var result dto.TemperatureSeriesResponse
	var err error
	if confirm {
		result, err = handler.service.Confirm(c.Request.Context(), id, request, actor)
	} else {
		result, err = handler.service.Invalidate(c.Request.Context(), id, request, actor)
	}
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}
