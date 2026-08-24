package handler

import (
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/service"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

type StrengthForecastHandler struct {
	service service.StrengthForecastService
}

func NewStrengthForecastHandler(value service.StrengthForecastService) *StrengthForecastHandler {
	return &StrengthForecastHandler{service: value}
}

func (handler *StrengthForecastHandler) List(c *gin.Context) {
	page, size := util.Pagination(c)
	sectionID, _ := strconv.ParseUint(c.Query("pour_section_id"), 10, 64)
	seriesID, _ := strconv.ParseUint(c.Query("temperature_series_id"), 10, 64)
	result, err := handler.service.List(c.Request.Context(), dto.StrengthForecastQuery{
		PourSectionID: uint(sectionID), TemperatureSeriesID: uint(seriesID),
		ForecastState: c.Query("forecast_state"), Page: page, PageSize: size,
	})
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}

func (handler *StrengthForecastHandler) Get(c *gin.Context) {
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

func (handler *StrengthForecastHandler) Run(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.RunStrengthForecastRequest
	if !util.BindJSON(c, &request) {
		return
	}
	result, created, err := handler.service.Run(c.Request.Context(), request, c.GetHeader("Idempotency-Key"), actor)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	if created {
		util.Created(c, result)
	} else {
		util.OK(c, result)
	}
}

func (handler *StrengthForecastHandler) Review(c *gin.Context)  { handler.action(c, "review") }
func (handler *StrengthForecastHandler) Confirm(c *gin.Context) { handler.action(c, "confirm") }
func (handler *StrengthForecastHandler) Void(c *gin.Context)    { handler.action(c, "void") }

func (handler *StrengthForecastHandler) action(c *gin.Context, action string) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	var request dto.ForecastActionRequest
	if !util.BindJSON(c, &request) {
		return
	}
	var result dto.StrengthForecastResponse
	var err error
	switch action {
	case "review":
		result, err = handler.service.Review(c.Request.Context(), id, request, actor)
	case "confirm":
		result, err = handler.service.Confirm(c.Request.Context(), id, request, actor)
	default:
		result, err = handler.service.Void(c.Request.Context(), id, request, actor)
	}
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}

func (handler *StrengthForecastHandler) Replay(c *gin.Context) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	result, err := handler.service.Replay(c.Request.Context(), id, actor)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}

func (handler *StrengthForecastHandler) Compare(c *gin.Context) {
	id, ok := util.ParseID(c, "id")
	if !ok {
		return
	}
	otherID, ok := util.ParseID(c, "other_id")
	if !ok {
		return
	}
	result, err := handler.service.Compare(c.Request.Context(), id, otherID)
	if err != nil {
		util.WriteError(c, err)
		return
	}
	util.OK(c, result)
}
