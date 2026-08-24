package middleware

import (
	"concrete-curing-maturity-monitor/backend/internal/model"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Claims struct {
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	jwt.RegisteredClaims
}

type AuthSession struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserView  `json:"user"`
}

type UserView struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type Authenticator struct {
	db     *gorm.DB
	secret []byte
	expiry time.Duration
}

func NewAuthenticator(db *gorm.DB, secret string, expiry time.Duration) *Authenticator {
	return &Authenticator{db: db, secret: []byte(secret), expiry: expiry}
}

func (auth *Authenticator) Login(ctx context.Context, username, password string) (AuthSession, error) {
	var user model.User
	if err := auth.db.WithContext(ctx).Where("username = ?", strings.TrimSpace(username)).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return AuthSession{}, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "username or password is incorrect")
		}
		return AuthSession{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load account", err)
	}
	if !user.Active || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return AuthSession{}, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "username or password is incorrect")
	}
	now := time.Now().UTC()
	expiresAt := now.Add(auth.expiry)
	claims := Claims{
		UserID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "concrete-curing-maturity-monitor", Subject: fmt.Sprintf("%d", user.ID),
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(auth.secret)
	if err != nil {
		return AuthSession{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to issue access token", err)
	}
	return AuthSession{
		Token: token, ExpiresAt: expiresAt,
		User: UserView{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role},
	}, nil
}

func (auth *Authenticator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			util.WriteError(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "a bearer token is required"))
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing algorithm %s", token.Method.Alg())
			}
			return auth.secret, nil
		})
		if err != nil || !token.Valid {
			util.WriteError(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "the access token is invalid or expired"))
			return
		}
		var user model.User
		if err := auth.db.WithContext(c.Request.Context()).First(&user, claims.UserID).Error; err != nil || !user.Active {
			util.WriteError(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "the account is unavailable"))
			return
		}
		actor := util.Actor{UserID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Role: user.Role, RequestID: util.RequestID(c)}
		c.Set(util.ActorContextKey, actor)
		c.Next()
	}
}
