package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	cookieName  = "sub-store-session"
	cookiePath  = "/"
	sessionTTL  = 7 * 24 * time.Hour // 7 days
)

type session struct {
	createdAt time.Time
}

// Manager holds credentials and the live session table.
type Manager struct {
	username string
	password string

	mu       sync.Mutex
	sessions map[string]*session // token → session
}

func New(username, password string) *Manager {
	return &Manager{
		username: username,
		password: password,
		sessions: make(map[string]*session),
	}
}

// GeneratePassword creates a random 16-char hex password for first-run.
func GeneratePassword() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// newToken creates a cryptographically random session token.
func newToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// RegisterAuthRoutes adds POST /api/auth/login and POST /api/auth/logout.
func (m *Manager) RegisterAuthRoutes(r *gin.Engine) {
	a := r.Group("/api/auth")
	a.POST("/login", m.handleLogin)
	a.POST("/logout", m.handleLogout)
	a.GET("/me", m.Middleware(), m.handleMe)
}

func (m *Manager) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Constant-time comparison to prevent timing attacks
	uOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(m.username)) == 1
	pOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(m.password)) == 1
	if !uOK || !pOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token := newToken()
	m.mu.Lock()
	m.sessions[token] = &session{createdAt: time.Now()}
	m.mu.Unlock()

	// HttpOnly cookie — JS cannot read it, protects against XSS
	c.SetCookie(cookieName, token, int(sessionTTL.Seconds()), cookiePath, "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (m *Manager) handleLogout(c *gin.Context) {
	if token, err := c.Cookie(cookieName); err == nil {
		m.mu.Lock()
		delete(m.sessions, token)
		m.mu.Unlock()
	}
	c.SetCookie(cookieName, "", -1, cookiePath, "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (m *Manager) handleMe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"username": m.username})
}

// Middleware returns a Gin handler that rejects unauthenticated requests.
func (m *Manager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || !m.validToken(token) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (m *Manager) validToken(token string) bool {
	if token == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[token]
	if !ok {
		return false
	}
	if time.Since(s.createdAt) > sessionTTL {
		delete(m.sessions, token)
		return false
	}
	return true
}

// GC removes expired sessions. Call periodically if needed (optional).
func (m *Manager) GC() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for token, s := range m.sessions {
		if time.Since(s.createdAt) > sessionTTL {
			delete(m.sessions, token)
		}
	}
}
