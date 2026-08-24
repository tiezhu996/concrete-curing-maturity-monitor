package config

import (
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/model"
	"concrete-curing-maturity-monitor/backend/internal/timeseries"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Port                   string
	DBDriver               string
	DBDSN                  string
	JWTSecret              string
	JWTExpiry              time.Duration
	ShutdownTimeout        time.Duration
	LoginLimitPerMinute    int
	ImportLimitPerMinute   int
	ForecastLimitPerMinute int
	MaxMissingRatio        float64
}

func Load() (Config, error) {
	expiry, err := time.ParseDuration(env("JWT_EXPIRY", "8h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_EXPIRY: %w", err)
	}
	shutdown, err := time.ParseDuration(env("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse SHUTDOWN_TIMEOUT: %w", err)
	}
	missing, err := strconv.ParseFloat(env("MAX_MISSING_RATIO", "0.10"), 64)
	if err != nil || missing < 0 || missing > 1 {
		return Config{}, fmt.Errorf("MAX_MISSING_RATIO must be between 0 and 1")
	}
	secret := env("JWT_SECRET", "")
	if len(secret) < 16 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 16 characters")
	}
	return Config{
		Port: env("PORT", "8080"), DBDriver: env("DB_DRIVER", "postgres"),
		DBDSN:     env("DB_DSN", "host=localhost user=curing_maturity password=curing_maturity_dev dbname=curing_maturity port=5432 sslmode=disable TimeZone=Asia/Shanghai"),
		JWTSecret: secret, JWTExpiry: expiry, ShutdownTimeout: shutdown,
		LoginLimitPerMinute:    intEnv("LOGIN_LIMIT_PER_MINUTE", 30),
		ImportLimitPerMinute:   intEnv("IMPORT_LIMIT_PER_MINUTE", 30),
		ForecastLimitPerMinute: intEnv("FORECAST_LIMIT_PER_MINUTE", 60),
		MaxMissingRatio:        missing,
	}, nil
}

func OpenDatabase(config Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch config.DBDriver {
	case "postgres":
		dialector = postgres.Open(config.DBDSN)
	case "sqlite":
		dialector = sqlite.Open(config.DBDSN)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q", config.DBDriver)
	}
	database, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", config.DBDriver, err)
	}
	if err := database.AutoMigrate(
		&model.User{}, &model.MixDesign{}, &model.PourSection{},
		&model.TemperatureSeries{}, &model.StrengthForecast{}, &model.AuditLog{},
	); err != nil {
		return nil, fmt.Errorf("migrate database schema: %w", err)
	}
	if err := seed(database); err != nil {
		return nil, fmt.Errorf("seed demonstration data: %w", err)
	}
	return database, nil
}

func seed(database *gorm.DB) error {
	return database.Transaction(func(tx *gorm.DB) error {
		users := []struct {
			Username, Password, DisplayName, Role string
		}{
			{"admin", "admin123", "System Administrator", constants.RoleAdmin},
			{"lab", "lab123", "Lab Engineer", constants.RoleLabEngineer},
			{"site", "site123", "Site Engineer", constants.RoleSiteEngineer},
			{"reviewer", "reviewer123", "Independent Reviewer", constants.RoleReviewer},
			{"auditor", "auditor123", "Quality Auditor", constants.RoleAuditor},
		}
		for _, candidate := range users {
			var count int64
			if err := tx.Model(&model.User{}).Where("username = ?", candidate.Username).Count(&count).Error; err != nil {
				return fmt.Errorf("check seed user %s: %w", candidate.Username, err)
			}
			if count > 0 {
				continue
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(candidate.Password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash seed password: %w", err)
			}
			user := model.User{Username: candidate.Username, PasswordHash: string(hash), DisplayName: candidate.DisplayName, Role: candidate.Role, Active: true}
			if err := tx.Create(&user).Error; err != nil {
				return fmt.Errorf("create seed user %s: %w", candidate.Username, err)
			}
		}

		var mixCount int64
		if err := tx.Model(&model.MixDesign{}).Count(&mixCount).Error; err != nil {
			return fmt.Errorf("count mix designs: %w", err)
		}
		if mixCount > 0 {
			return nil
		}
		var lab model.User
		if err := tx.Where("username = ?", "lab").First(&lab).Error; err != nil {
			return fmt.Errorf("load lab seed user: %w", err)
		}
		calibration, _ := json.Marshal([]map[string]float64{
			{"maturity_degree_hours": 0, "strength_mpa": 0},
			{"maturity_degree_hours": 120, "strength_mpa": 8.5},
			{"maturity_degree_hours": 360, "strength_mpa": 18.2},
			{"maturity_degree_hours": 720, "strength_mpa": 28.5},
			{"maturity_degree_hours": 1200, "strength_mpa": 36.8},
		})
		now := time.Now().UTC()
		validFrom := now.AddDate(0, -2, 0)
		mix := model.MixDesign{
			MixCode: "C35-P42", Version: 3, CementType: "CEM II/A-L 42.5 R",
			WaterBinderRatio: 0.42, DatumTemperatureC: 0,
			CalibrationPointsJSON: string(calibration), ValidFrom: &validFrom,
			DesignState: constants.MixPublished, CreatedBy: lab.ID,
			CreatedByName: lab.DisplayName, LockVersion: 1,
		}
		if err := tx.Create(&mix).Error; err != nil {
			return fmt.Errorf("create seed mix design: %w", err)
		}
		pouredAt := now.Add(-14 * time.Hour)
		section := model.PourSection{
			SectionCode: "B2-W07", Name: "Basement wall lift 07", StructurePart: "North retaining wall",
			VolumeM3: 86.4, MixDesignID: mix.ID, PouredAt: &pouredAt,
			TargetStrengthMPA: 28, CuringState: string(constants.CuringActive),
			OwnerTeam: "Civil works A", Version: 1,
		}
		if err := tx.Create(&section).Error; err != nil {
			return fmt.Errorf("create seed pour section: %w", err)
		}
		points := make([]timeseries.Point, 0, 8)
		for index, temperature := range []float64{21.4, 23.8, 27.1, 30.2, 31.6, 30.8, 29.4, 28.1} {
			points = append(points, timeseries.Point{Timestamp: pouredAt.Add(time.Duration(index) * 2 * time.Hour), TemperatureC: temperature})
		}
		encoded, checksum, err := timeseries.CanonicalJSONAndChecksum(points)
		if err != nil {
			return err
		}
		series := model.TemperatureSeries{
			PourSectionID: section.ID, SensorCode: "TC-B2-W07-A", SampleIntervalMin: 120,
			PointsJSON: encoded, StartedAt: points[0].Timestamp, EndedAt: points[len(points)-1].Timestamp,
			SourceChecksum: checksum, MissingRatio: 0, SeriesState: constants.SeriesUsable,
			QualityNote: "Seeded chronological series with complete two-hour intervals",
			ImportedBy:  lab.ID, ImportedByName: lab.DisplayName,
		}
		if err := tx.Create(&series).Error; err != nil {
			return fmt.Errorf("create seed temperature series: %w", err)
		}
		return nil
	})
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func intEnv(name string, fallback int) int {
	value, err := strconv.Atoi(env(name, strconv.Itoa(fallback)))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
