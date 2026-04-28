// controller/user.go

package controller

import (
	"errors"
	"fmt"
	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Username string `json:"username" binding:"omitempty"`
	Password string `json:"password" binding:"required"`
}

func clearCookies(c *gin.Context) {
	c.SetCookie(common.JwtCookieName, "", -1, "/", "", false, true)
	c.SetCookie("email", "", -1, "/", "", false, true)
}

//
//func Login(c *gin.Context) {
//	var req loginRequest
//	var user model.User
//	var err error
//	clientIP := c.ClientIP()
//
//	if err := c.ShouldBindJSON(&req); err != nil {
//		common.LogError(c.Request.Context(), "Login request binding error: "+err.Error())
//		c.JSON(400, gin.H{"error": "Invalid request"})
//		return
//	}
//
//	token, err := c.Cookie(common.JwtCookieName)
//	if err == nil && token != "" { // 已經登入了
//		c.JSON(200, gin.H{"message": "Already logged in"})
//		return
//	}
//
//	switch {
//	case req.Email != "":
//		user, err = model.LoginByEmail(req.Email)
//	case req.Username != "":
//		user, err = model.LoginByName(req.Username)
//	default:
//		c.JSON(400, gin.H{"error": "Email or Username is required"})
//		return
//	}
//
//	if err != nil {
//		if err.Error() == "record not found" {
//			c.JSON(401, gin.H{"error": "User not found"})
//		} else {
//			c.JSON(500, gin.H{"error": "Internal server error"})
//		}
//		_ = model.RecordAttempt(clientIP, false)
//		return
//	}
//
//	v := common.ValidatePasswordAndHash(req.Password+user.Salt, user.Password)
//
//	if !v {
//		_ = model.RecordAttempt(clientIP, false)
//		c.JSON(401, gin.H{"error": "Invalid password"})
//		return
//	}
//	UA := c.GetHeader("User-Agent")
//	now := time.Now()
//	msg := fmt.Sprintf(
//		"User: %s, IP: %s, User-Agent: %s login at %s",
//		user.Username,
//		clientIP,
//		UA,
//		now.Format("2006/01/02-15:04:05"),
//	)
//	common.LogDebug(c.Request.Context(), msg)
//	_ = model.ReSetFail(clientIP)
//	clientDeviceID, err := c.Cookie("device_id")
//	if err != nil {
//		clientDeviceID = common.GenerateDeviceIDWithIP(clientIP)
//		c.SetCookie( // 先寫進餅乾裡 下次直接取
//			"device_id",    // name
//			clientDeviceID, // value
//			60*60*24*365,   // maxAge (秒)：一年
//			"/",            // path
//			"",             // domain (留空為當前 host)
//			false,          // secure (https 才送)
//			true,           // httpOnly
//		)
//		CreateVerificationCode(c, user) // 發送驗證碼
//		c.JSON(202, gin.H{"message": "verification code sent, for new device"})
//		return
//	}
//	// 檢查是否已存在此裝置的登入紀錄
//	isExists, dberr := model.IsDeviceExists(clientDeviceID)
//	if !isExists || dberr != nil { //不存在就跑email 驗證
//		CreateVerificationCode(c, user) // 發送驗證碼
//		c.JSON(202, gin.H{"message": "verification code sent"})
//		return
//	}
//	// 如果存在就直接登入
//
//	SetUpJWT(c, user) // 設置 JWT
//
//}

// 只剩client app 再用 過陣子刪除
func CreateVerificationCode(c *gin.Context, user model.User) {
	common.LogDebug(c.Request.Context(), "CreateVerificationCode called for user: "+user.Username)
	c.SetCookie(
		"email",
		user.Email, // 使用者的 email
		60*5,
		"/",   // path
		"",    // domain (留空為當前 host)
		false, // secure (https 才送)
		true,  // httpOnly
	)
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	err := model.SetVerificationCode(user.ID, code)
	if err != nil {
		if err.Error() == "verification code already set and not expired" {
			return
		}
		common.LogError(c.Request.Context(), "SetVerificationCode error: "+err.Error())
		return
	}

	htmlMsg := fmt.Sprintf(
		`<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>Verification Code</title>
	<style>
		body {
		margin: 0;
		padding: 0;
		font-family: sans-serif;
		line-height: 1.4;
		background: linear-gradient(-45deg,
			#ff0000, #ff7f00, #ffff00, #00ff00, #0000ff, #4b0082, #8f00ff);
		background-size: 400%% 400%%;
		animation: rainbow 15s ease infinite;
		}
		@keyframes rainbow {
		0%%   { background-position: 0%% 50%%; }
		50%%  { background-position: 100%% 50%%; }
		100%% { background-position: 0%% 50%%; }
		}
		.container {
		padding: 20px;
		background: rgba(255, 255, 255, 0.8);
		margin: 40px auto;
		max-width: 600px;
		border-radius: 8px;
		}
		p { margin: 1em 0; }
	</style>
	</head>
	<body>
	<div class="container">
		<p>Hello %s,</p>
		<p>Your verification code is: <strong>%s</strong></p>
		<p>
		Please use this code to verify your login. 
		This code will expire in <strong>5 minutes</strong>.
		</p>
		<p>If you did not request this, please ignore this email.</p>
		<br>
		<p>Thank you,<br>The %s Team</p>
	</div>
	</body>
	</html>`,
		user.DisplayName,
		code,
		common.SystemName,
	)

	email := user.Email

	if email == "" || email == "null" {
		common.LogError(c.Request.Context(), "User email is empty for user: "+user.Username)
	}

	common.LogDebug(c.Request.Context(), "Sending verification code to user: "+user.Username+" Email: "+email)

	err = common.SendEmail(
		"Login Verification Code",
		user.Email, // 使用者的 email
		htmlMsg,
	)

	if err != nil {
		common.LogError(c.Request.Context(), "SendEmail error: "+err.Error()+" SMTP account: "+common.SMTPAccount)
		common.LogError(c.Request.Context(), "Failed to send verification code to user: "+user.Username)
	}
}

type VerifyLoginCache struct {
	Code     string
	Email    string
	Exp      time.Duration
	CreateAt time.Time
}

type VerifyTokenCache struct {
	AuthToken string
	UserEmail string
	Exp       time.Duration
	CreateAt  time.Time
}

type EmailChallengeStore struct {
	cache      map[string]VerifyLoginCache
	tokenCache map[string]VerifyTokenCache
	Mu         sync.RWMutex
}

func NewChallengeStore() *EmailChallengeStore {
	return &EmailChallengeStore{
		cache:      make(map[string]VerifyLoginCache),
		tokenCache: make(map[string]VerifyTokenCache),
	}
}

func (s *EmailChallengeStore) get(id string) *VerifyLoginCache {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	v, _ := s.cache[id]
	return &v
}

func (s *EmailChallengeStore) DelById(id string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	_, ok := s.cache[id]
	if !ok {
		return
	}
	delete(s.cache, id)
}

func (s *EmailChallengeStore) getEmail(id string) (string, bool) {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	v, ok := s.cache[id]
	if !ok {
		return "", false
	}

	return v.Email, true
}

func (s *EmailChallengeStore) valid(id string, inpCode string, hashEmail string) bool {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	v, ok := s.cache[id]
	if !ok {
		return false
	}

	ca := v.CreateAt
	expired := time.Now().After(ca.Add(v.Exp))
	isCode := inpCode == v.Email
	validHash := common.ValidatePasswordAndHash(hashEmail, v.Email)
	if expired && !isCode && !validHash {
		return false
	}
	delete(s.cache, id)

	return true
}

func (s *EmailChallengeStore) CreateLoginChallenge(id, email string) string {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if v, ok := s.cache[id]; ok {
		isExpired := time.Now().After(v.CreateAt.Add(v.Exp))
		if !isExpired {
			return v.Code
		}
		delete(s.cache, id)
	}

	code := common.GetRandomIntString(16)
	s.cache[id] = VerifyLoginCache{
		Code:     code,
		Email:    email,
		CreateAt: time.Now(),
		Exp:      5 * time.Minute,
	}

	return code
}

func (s *EmailChallengeStore) setVerifyToken(id, code, email string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	s.tokenCache[id] = VerifyTokenCache{
		AuthToken: code,
		UserEmail: email,
		Exp:       3 * time.Minute,
		CreateAt:  time.Now(),
	}
}

func (s *EmailChallengeStore) UrlVerifyLogin(c *gin.Context) {
	clientIP := c.ClientIP() // for login Attempt Record

	code := c.Query("code")
	hashEmail := c.Query("eh")
	id := c.Query("id")

	email, ok := s.getEmail(id)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not found"})
		return
	}
	isValid := s.valid(id, code, hashEmail)

	if !isValid {
		_ = model.RecordAttempt(clientIP, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired verification code"})
		return
	}
	_ = model.RecordAttempt(clientIP, true)
	frontend := common.GetEnvOrDefaultString("FRONTEND_BASE_URL", "http://localhost:3000")
	authCode := common.GetRandomString(32)
	uri := fmt.Sprintf("%s/login/callback?code=%s?=id%s", frontend, authCode, id)

	s.setVerifyToken(id, authCode, email)

	c.Redirect(http.StatusFound, uri)
	return
}

const (
	serverErr int8 = 1
	authErr   int8 = 0
	nilErr    int8 = 99
)

func (s *EmailChallengeStore) challenge(id, code, ip string) (string, int8, error) {
	s.Mu.Lock()

	v, ok := s.tokenCache[id]
	if !ok {
		s.Mu.Unlock()
		return "", authErr, fmt.Errorf("not found")
	}
	ca := v.CreateAt
	expired := time.Now().After(ca.Add(v.Exp))
	if expired {
		delete(s.tokenCache, id)
		s.Mu.Unlock()
		return "", authErr, fmt.Errorf("expired")
	}
	if code != v.AuthToken {
		delete(s.tokenCache, id)
		s.Mu.Unlock()
		return "", authErr, fmt.Errorf("invalid verification code")
	}
	logger.Debugf("Verification code valid for id: %s, email: %s", id, v.UserEmail)
	email := v.UserEmail
	delete(s.tokenCache, id)
	s.Mu.Unlock()

	exp := time.Now().Add(common.JwtExpireSeconds * time.Second).Unix()

	user, err := model.GetUserByEmail(email)
	if err != nil {
		return "", serverErr, err
	}

	payload := map[string]interface{}{
		"user_id":  fmt.Sprint(user.ID),
		"username": user.Username,
		"role":     user.Role,
		"Login_IP": ip,
		"exp":      exp,
	}

	token, err := common.GenerateJWTToken(payload)
	if err != nil {
		return "", serverErr, err
	}

	return token, nilErr, nil
}

func (s *EmailChallengeStore) ExchangeToken(c *gin.Context) {
	code := c.Query("code")
	id := c.Query("id")
	ip := c.ClientIP()
	token, errCode, err := s.challenge(id, code, ip)
	if err != nil {
		if errCode == serverErr {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	//
	//c.SetCookie(
	//	common.JwtCookieName,    // name
	//	token,                   // value
	//	common.JwtExpireSeconds, // maxAge
	//	"/",                     // path
	//	"",                      // domain (留空為當前 host)
	//	false,                   // secure (https 才送)
	//	true,                    // httpOnly
	//)

	c.JSON(http.StatusOK, gin.H{"token": token})
	return
}

func (s *EmailChallengeStore) ChallengeLogin(c *gin.Context) {
	clientIP := c.ClientIP()
	var req loginRequest
	var user model.User
	var err error
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch {
	case req.Email != "":
		user, err = model.LoginByEmail(req.Email)
	case req.Username != "":
		user, err = model.LoginByName(req.Username)
	default:
		c.JSON(400, gin.H{"error": "Email or Username is required"})
		return
	}

	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(401, gin.H{"error": "User not found"})
		} else {
			c.JSON(500, gin.H{"error": "Internal server error"})
		}
		_ = model.RecordAttempt(clientIP, false)
		return
	}

	v := common.ValidatePasswordAndHash(req.Password+user.Salt, user.Password)
	if !v {
		c.JSON(401, gin.H{"error": "Invalid password"})
		return
	}

	if err := s.VerificationEmail(c, user.Email, fmt.Sprint(user.ID), user.Username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Send Email For new device."})
	return
}

func (s *EmailChallengeStore) VerificationEmail(c *gin.Context, email, id, username string) error {
	hashEmail, err := common.Password2Hash(email)
	if err != nil {
		return err
	}
	code := s.CreateLoginChallenge(id, email)
	defaultUrl := "http://localhost:" + common.GetEnvOrDefaultString("PORT", "8080")
	url := fmt.Sprintf("%s/Authentication/verify?code=%s&eh=%s&id=%s", defaultUrl, code, hashEmail, id)
	htmlMsg := fmt.Sprintf(
		`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Verification Link</title>
  <style>
    body {
      margin: 0;
      padding: 24px 12px;
      font-family: "Segoe UI", "Noto Sans TC", Arial, sans-serif;
      line-height: 1.6;
      background: radial-gradient(circle at center, #1a1a20 0%%, #0a0a0c 100%%);
      color: #e5e7eb;
    }
    .container {
      max-width: 620px;
      margin: 0 auto;
      padding: 24px;
      border-radius: 10px;
      background: rgba(20, 20, 25, 0.92);
      border: 1px solid #333333;
      box-shadow: 0 0 24px rgba(0, 0, 0, 0.45), inset 0 0 16px rgba(24, 160, 88, 0.08);
    }
    .title {
      margin: 0 0 14px 0;
      font-size: 20px;
      color: #18a058;
      letter-spacing: 0.4px;
      font-weight: 700;
    }
    p {
      margin: 10px 0;
      color: #cbd5e1;
    }
    .verify-btn {
      display: inline-block;
      margin: 10px 0 4px 0;
      padding: 10px 16px;
      border-radius: 6px;
      border: 1px solid #18a058;
      background: #18a058;
      color: #ffffff !important;
      text-decoration: none;
      font-weight: 700;
      letter-spacing: 0.4px;
    }
    .muted {
      color: #9ca3af;
      font-size: 13px;
      margin-top: 14px;
    }
    .mono {
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 12px;
      color: #cbd5e1;
      word-break: break-all;
      background: rgba(0, 0, 0, 0.25);
      border: 1px solid #2b2b2b;
      padding: 10px;
      border-radius: 6px;
    }
  </style>
</head>
<body>
  <div class="container">
    <h1 class="title">MC-SERVER Verification</h1>
    <p>Hello %s,</p>
    <p>Click the button below to verify your login:</p>

    <p>
      <a class="verify-btn" href="%s" target="_blank" rel="noopener noreferrer">
        VERIFY LOGIN
      </a>
    </p>

    <p>This link will expire in <strong>5 minutes</strong>.</p>

    <p class="muted">If the button does not work, copy this URL:</p>
    <p class="mono">%s</p>

    <p>If you did not request this, please ignore this email.</p>
    <p>Thank you,<br>The %s Team</p>
  </div>
</body>
</html>`,
		username,
		url,
		url,
		common.SystemName,
	)

	err = common.SendEmail(
		"Login Verification Code",
		email, // 使用者的 email
		htmlMsg,
	)

	if err != nil {
		common.LogError(c.Request.Context(), "SendEmail error: "+err.Error()+" SMTP account: "+common.SMTPAccount)
		common.LogError(c.Request.Context(), "Failed to send verification code to user: "+username)
	}

	return err
}

//func VerifyLogin(c *gin.Context) {
//	var req struct {
//		Code string `json:"code" binding:"required"`
//	}
//	clientIP := c.ClientIP()
//	if err := c.ShouldBindJSON(&req); err != nil {
//		common.LogError(c.Request.Context(), "VerifyLogin request binding error: "+err.Error())
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid request"})
//		return
//	}
//
//	_, err := c.Cookie(common.JwtCookieName)
//	if err == nil { // 已經登入了
//		c.JSON(200, gin.H{"message": "Already logged in"})
//		return
//	}
//
//	email, cookieErr := c.Cookie("email")
//	if cookieErr != nil {
//		common.LogError(c.Request.Context(), "Failed to get email from cookie: "+err.Error())
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request."})
//		return
//	}
//
//	code, sendAt, ver_err := model.GetVerificationCode(email)
//	if ver_err != nil {
//		common.LogError(c.Request.Context(), "Verify Error"+err.Error())
//		_ = model.RecordAttempt(clientIP, false)
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
//		return
//	}
//
//	if code != req.Code || time.Since(sendAt) > 5*time.Minute {
//		_ = model.RecordAttempt(clientIP, false)
//		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired verification code"})
//		return
//	}
//	_ = model.ReSetFail(clientIP)
//	_ = model.ClearVerificationCode(email) // 重置驗證碼
//	user, _ := model.GetUserByEmail(email)
//
//	appHeadertDeviceID := c.GetHeader(common.ClientHeader)
//	ua := strings.ToLower(c.GetHeader("User-Agent"))
//	if strings.Contains(ua, "mpmc client ua") && appHeadertDeviceID != "" {
//		SetUpAppJWT(c, user)
//		return
//	}
//
//	SetUpJWT(c, user)
//}
//
//func SetUpJWT(c *gin.Context, user model.User) {
//
//	exp := time.Now().Add(common.JwtExpireSeconds * time.Second).Unix()
//
//	payload := map[string]interface{}{
//		"user_id":  fmt.Sprint(user.ID),
//		"username": user.Username,
//		"role":     user.Role,
//		"Login_IP": c.ClientIP(),
//		"exp":      exp,
//	}
//
//	t, err := common.GenerateJWTToken(payload)
//
//	if err != nil {
//		common.LogError(c.Request.Context(), "GenerateJWTToken error: "+err.Error())
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
//		return
//	}
//
//	clientDeviceID, _ := c.Cookie("device_id")
//
//	ua := c.GetHeader("User-Agent")
//
//	ip := c.ClientIP()
//
//	model.SaveDevice(
//		clientDeviceID,
//		ua,
//		ip,
//		user.ID)
//
//	c.SetCookie(
//		common.JwtCookieName,    // name
//		t,                       // value
//		common.JwtExpireSeconds, // maxAge
//		"/",                     // path
//		"",                      // domain (留空為當前 host)
//		false,                   // secure (https 才送)
//		true,                    // httpOnly
//	)
//
//	c.SetCookie("email", "", -1, "/", "", false, true)
//	c.JSON(http.StatusOK, gin.H{
//		"message": "Login successful"})
//}

func tokenInfo(token string) (string, time.Time, error) {
	var userId string
	var exp time.Time
	payload, err := common.GetJWTPayload(token)
	if err != nil {
		common.Logger.Error("GetJWTPayload error: " + err.Error())
		return "", time.Now(), errors.New("invalid token")
	}
	expRaw, ok := payload["exp"]
	if ok {
		expFloat, ok := expRaw.(float64)
		if !ok {
			exp = time.Now().Add(common.JwtExpireSeconds * time.Second / 2)
		}
		exp = time.Unix(int64(expFloat), 0)
	}
	userId = ""
	if rawUID, ok := payload["user_id"]; ok {
		userId = fmt.Sprint(rawUID)
	}
	return userId, exp, nil
}

func Logout(c *gin.Context) {
	token := c.Request.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	user, exp, err := tokenInfo(token)
	if err != nil {
		common.LogError(c.Request.Context(), "Logout error: "+err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}
	common.RevokedTokens.Add(token, user, exp)
	// 清除 JWT Cookie
	clearCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

type AmongUs struct {
	agm *service.GameManager
}

func NewAmongUsController(agm *service.GameManager) *AmongUs {
	return &AmongUs{agm: agm}
}

func (a *AmongUs) Join(c *gin.Context) {
	gameId := c.Param("id")
	if gameId == "" {
		c.JSON(400, gin.H{"error": "傻逼"})
		return
	}

	clientIP := c.ClientIP()

	clientDeviceID, err := c.Cookie("device_id")
	if err != nil {
		clientDeviceID = common.GenerateDeviceIDWithIP(clientIP)
		c.SetCookie( // 先寫進餅乾裡 下次直接取
			"device_id",    // name
			clientDeviceID, // value
			60*60*24*365,   // maxAge (秒)：一年
			"/",            // path
			"",             // domain (留空為當前 host)
			false,          // secure (https 才送)
			true,           // httpOnly
		)
		err = nil
	}

	role, task, taskInfo, rt, err := a.agm.Join(clientDeviceID, gameId)
	// role, task, taskInfo, rt, err := a.agm.Join(common.GetRandomString(8), gameId)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": role, "Task": task, "TaskInfo": taskInfo, "RoundTasks": rt})
}

// admin method
func (a *AmongUs) AllGames(c *gin.Context) {
	//admin method

	gs := a.agm.List()

	c.JSON(200, gin.H{
		"message": "all games",
		"games":   gs,
	})

}

func (a *AmongUs) Create(c *gin.Context) {
	//admin method

	num := c.Param("num")

	if num == "" {
		num = "5"
	}

	gs, err := a.agm.Create(num)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Create sussues", "game_id": gs.ID()})
}

func (a *AmongUs) EndGame(c *gin.Context) {
	//admin method

	gameId := c.Param("id")
	if gameId == "" {
		c.JSON(400, gin.H{"error": "傻逼"})
		return
	}

	err := a.agm.EndGame(gameId)

	if err != nil {
		c.JSON(400, gin.H{"error": "傻逼"})
		return
	}

	c.JSON(200, gin.H{"message": "del"})
}

func (a *AmongUs) ListPlayers(c *gin.Context) {
	gameId := c.Param("id")
	if gameId == "" {
		c.JSON(400, gin.H{"error": "傻逼"})
		return
	}

	players, err := a.agm.ListPlayers(gameId)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": players})

}
