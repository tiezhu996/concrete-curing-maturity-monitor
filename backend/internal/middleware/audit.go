package middleware

import (
	"concrete-curing-maturity-monitor/backend/internal/model"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuditRecorder struct{ db *gorm.DB }

func NewAuditRecorder(db *gorm.DB) *AuditRecorder { return &AuditRecorder{db: db} }

func (recorder *AuditRecorder) Record(
	ctx context.Context,
	actor util.Actor,
	entityType string,
	entityID uint,
	action string,
	before any,
	after any,
	metadata any,
) error {
	beforeJSON, err := encodeAuditValue(before)
	if err != nil {
		return fmt.Errorf("encode audit before snapshot: %w", err)
	}
	afterJSON, err := encodeAuditValue(after)
	if err != nil {
		return fmt.Errorf("encode audit after snapshot: %w", err)
	}
	metadataJSON, err := encodeAuditValue(metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	entry := model.AuditLog{
		ActorID: actor.UserID, ActorName: actor.DisplayName, ActorRole: actor.Role,
		RequestID: actor.RequestID, EntityType: entityType, EntityID: entityID,
		Action: action, BeforeJSON: beforeJSON, AfterJSON: afterJSON,
		MetadataJSON: metadataJSON, CreatedAt: time.Now().UTC(),
	}
	defer func() {
		_ = recorder.db.WithContext(ctx).Create(&entry).Error
	}()
	return nil
}

func AuditContext(recorder *AuditRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("audit_recorder", recorder)
		c.Next()
	}
}

func (recorder *AuditRecorder) ListHandler(c *gin.Context) {
	database := recorder.db.WithContext(c.Request.Context()).Model(&model.AuditLog{})
	if value := c.Query("entity_type"); value != "" {
		database = database.Where("entity_type = ?", value)
	}
	if value := c.Query("actor_id"); value != "" {
		if id, err := strconv.ParseUint(value, 10, 64); err == nil {
			database = database.Where("actor_id = ?", id)
		}
	}
	if value := c.Query("request_id"); value != "" {
		database = database.Where("request_id = ?", value)
	}
	if value := c.Query("from"); value != "" {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			database = database.Where("created_at >= ?", parsed)
		}
	}
	if value := c.Query("to"); value != "" {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			database = database.Where("created_at <= ?", parsed)
		}
	}
	page, size := util.Pagination(c)
	var total int64
	if err := database.Count(&total).Error; err != nil {
		util.WriteError(c, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to count audit logs", err))
		return
	}
	var entries []model.AuditLog
	if err := database.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&entries).Error; err != nil {
		util.WriteError(c, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list audit logs", err))
		return
	}
	util.OK(c, gin.H{"items": entries, "total": total, "page": page, "page_size": size})
}

func encodeAuditValue(value any) (string, error) {
	if value == nil {
		return "null", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
