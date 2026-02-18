package controller

import (
	"errors"
	"fmt"
	"go-backend/common"
	"go-backend/model"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type oidcLoginRequest struct {
	Provider string `json:"provider" binding:"required"`
	IDToken  string `json:"id_token" binding:"required"`
}

type oidcClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Subject       string `json:"sub"`
}

func OIDCLogin(c *gin.Context) {
	var req oidcLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.LogError(c.Request.Context(), "OIDC login request binding error: "+err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	providerName := strings.ToLower(strings.TrimSpace(req.Provider))
	provider, err := GetOIDCProvider(providerName)
	if err != nil {
		common.LogError(c.Request.Context(), "OIDC provider error: "+err.Error())
		switch {
		case errors.Is(err, ErrOIDCUnsupportedProvider):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
		case errors.Is(err, ErrOIDCProviderNotConfigured), errors.Is(err, ErrOIDCNotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Provider not configured"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "OIDC init failed"})
		}
		return
	}

	idToken, err := provider.Verifier.Verify(c.Request.Context(), req.IDToken)
	if err != nil {
		common.LogError(c.Request.Context(), "OIDC token verify failed: "+err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		common.LogError(c.Request.Context(), "OIDC token claims error: "+err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	claims.Email = strings.TrimSpace(claims.Email)
	claims.Subject = strings.TrimSpace(claims.Subject)

	if claims.Subject == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	if claims.Email == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email claim missing"})
		return
	}

	if common.OIDCRequireEmailVerified && !claims.EmailVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email not verified"})
		return
	}

	user, err := model.GetUserByEmail(claims.Email)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not found"})
		return
	}

	if err := model.UpsertUserIdentity(user.ID, provider.Name, claims.Subject, claims.Email); err != nil {
		common.LogError(c.Request.Context(), "UpsertUserIdentity error: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	token, err := issueJWTForUser(user, c.ClientIP())
	if err != nil {
		common.LogError(c.Request.Context(), "GenerateJWTToken error: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func issueJWTForUser(user model.User, ip string) (string, error) {
	exp := time.Now().Add(common.JwtExpireSeconds * time.Second).Unix()
	payload := map[string]interface{}{
		"user_id":  fmt.Sprint(user.ID),
		"username": user.Username,
		"role":     user.Role,
		"Login_IP": ip,
		"exp":      exp,
	}
	return common.GenerateJWTToken(payload)
}
