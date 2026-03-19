package ratelimit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	MaxFailures = 3
	BanDuration = 24 * time.Hour
	MinDelay    = 800 * time.Millisecond
	MaxDelay    = 1200 * time.Millisecond
	PersistFile = "ratelimit.json"
)

type IPRecord struct {
	Failures int       `json:"failures"`
	BannedAt time.Time `json:"banned_at,omitempty"`
	IsBanned bool      `json:"is_banned"`
}

type Store struct {
	mu      sync.Mutex
	records map[string]*IPRecord
	file    string
}

type persistData struct {
	Records map[string]*IPRecord `json:"records"`
	Date    string               `json:"date"`
}

func New(dataDir string) *Store {
	s := &Store{
		records: make(map[string]*IPRecord),
		file:    filepath.Join(dataDir, PersistFile),
	}
	s.load()
	return s
}

func (s *Store) load() {
	data, err := os.ReadFile(s.file)
	if err != nil {
		return
	}
	var pd persistData
	if err := json.Unmarshal(data, &pd); err != nil {
		return
	}
	today := time.Now().Format("2006-01-02")
	if pd.Date != today {
		// 新的一天，清空并删除旧文件
		_ = os.Remove(s.file)
		return
	}
	s.records = pd.Records
}

func (s *Store) save() {
	pd := persistData{
		Records: s.records,
		Date:    time.Now().Format("2006-01-02"),
	}
	data, _ := json.MarshalIndent(pd, "", "  ")
	_ = os.WriteFile(s.file, data, 0600)
}

func getIP(c *gin.Context) string {
	return c.ClientIP()
}

func (s *Store) isBanned(ip string) bool {
	rec, ok := s.records[ip]
	if !ok || !rec.IsBanned {
		return false
	}
	today := time.Now().Format("2006-01-02")
	bannedDay := rec.BannedAt.Format("2006-01-02")
	if today != bannedDay {
		delete(s.records, ip)
		s.save()
		return false
	}
	return true
}

func (s *Store) recordFailure(ip string) bool {
	rec, ok := s.records[ip]
	if !ok {
		rec = &IPRecord{}
		s.records[ip] = rec
	}
	rec.Failures++
	if rec.Failures >= MaxFailures {
		rec.IsBanned = true
		rec.BannedAt = time.Now()
		fmt.Printf("[Security] IP %s banned until tomorrow (%d failures)\n", ip, rec.Failures)
	}
	s.save()
	return rec.IsBanned
}

func (s *Store) recordSuccess(ip string) {
	if _, ok := s.records[ip]; ok {
		delete(s.records, ip)
		s.save()
	}
}

// fixedDelay 确保响应时间不低于 MinDelay，防止时序攻击
func fixedDelay(start time.Time) {
	elapsed := time.Since(start)
	if elapsed < MinDelay {
		time.Sleep(MinDelay - elapsed)
	}
}

// PathGuard 保护路径验证入口（/api/utils/env）
// 固定响应时间 + 失败封禁
func (s *Store) PathGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ip := getIP(c)

		s.mu.Lock()
		banned := s.isBanned(ip)
		s.mu.Unlock()

		if banned {
			fixedDelay(start)
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		c.Next()

		status := c.Writer.Status()
		s.mu.Lock()
		if status == http.StatusNotFound || status >= 400 {
			s.recordFailure(ip)
		} else if status == http.StatusOK {
			s.recordSuccess(ip)
		}
		s.mu.Unlock()

		fixedDelay(start)
	}
}

// SubGuard 保护 /sub/:token 订阅端点
func (s *Store) SubGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ip := getIP(c)

		s.mu.Lock()
		banned := s.isBanned(ip)
		s.mu.Unlock()

		if banned {
			fixedDelay(start)
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		c.Next()

		status := c.Writer.Status()
		s.mu.Lock()
		if status == http.StatusNotFound {
			s.recordFailure(ip)
		} else if status == http.StatusOK {
			s.recordSuccess(ip)
		}
		s.mu.Unlock()

		fixedDelay(start)
	}
}
