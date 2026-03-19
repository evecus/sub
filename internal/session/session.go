package session

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const cookieName = "ss_date"

// SetDaySession 在响应里设置当天的 session cookie，到 24:00 失效
func SetDaySession(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	maxAge := int(midnight.Sub(now).Seconds())

	c.SetCookie(cookieName, today, maxAge, "/", "", false, true)
}

// Guard 中间件：检查 Cookie 里的日期是否是今天，不是则 401
func Guard() gin.HandlerFunc {
	return func(c *gin.Context) {
		today := time.Now().Format("2006-01-02")
		val, err := c.Cookie(cookieName)
		if err != nil || val != today {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status": "failed",
				"error":  gin.H{"message": "session expired, please reconnect"},
			})
			return
		}
		c.Next()
	}
}
