package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/evecus/sub/internal/exporter"
	"github.com/evecus/sub/internal/parser"
	"github.com/evecus/sub/internal/store"
)

type Handler struct {
	store       *store.Store
	backendPath string
}

func New(s *store.Store, backendPath string) *Handler {
	return &Handler{store: s, backendPath: backendPath}
}

// RegisterRoutes mounts all management API (no auth — path is the secret).
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	// ── Utils (Sub-Store compat) ──────────────────────────────────────────────
	rg.GET("/utils/env", h.getEnv)
	rg.GET("/utils/refresh", h.refresh)

	// ── Subscriptions (Sub-Store compat: /api/subs) ───────────────────────────
	rg.GET("/subs", h.listSubs)
	rg.POST("/subs", h.createSub)
	rg.GET("/subs/:name", h.getOneSub)
	rg.PATCH("/subs/:name", h.editSub)
	rg.DELETE("/subs/:name", h.deleteSub)
	rg.PUT("/subs", h.sortSubs)

	// ── Collections (Sub-Store compat: /api/collections) ─────────────────────
	rg.GET("/collections", h.listCollections)
	rg.POST("/collections", h.createCollection)
	rg.GET("/collections/:name", h.getOneCollection)
	rg.PATCH("/collections/:name", h.editCollection)
	rg.DELETE("/collections/:name", h.deleteCollection)
	rg.PUT("/collections", h.sortCollections)

	// ── Share tokens (Sub-Store compat: /api/token, /api/tokens) ─────────────
	rg.POST("/token", h.createToken)
	rg.GET("/tokens", h.listTokens)
	rg.DELETE("/token/:token", h.deleteToken)

	// ── Flow info (stub — we don't have real flow data) ───────────────────────
	rg.GET("/sub/flow/:name", h.getFlow)

	// ── Preview (stub) ────────────────────────────────────────────────────────
	rg.POST("/preview/subs", h.previewSub)
	rg.POST("/preview/collections", h.previewSub)

	// ── Sort ──────────────────────────────────────────────────────────────────
	rg.POST("/sort/subs", h.sortSubs)
	rg.POST("/sort/collections", h.sortCollections)
	rg.POST("/sort/tokens", h.sortTokens)

	// ── Files (stub — return empty) ────────────────────────────────────────────
	rg.GET("/wholeFiles", h.listFiles)
	rg.GET("/files", h.listFiles)

	// ── Settings (stub) ───────────────────────────────────────────────────────
	rg.GET("/settings", h.getSettings)
	rg.PATCH("/settings", h.patchSettings)

	// ── Artifacts (stub) ─────────────────────────────────────────────────────
	rg.GET("/artifacts", h.listArtifacts)

	// ── Node info (stub) ─────────────────────────────────────────────────────
	rg.POST("/utils/node-info", h.nodeInfo)
}

// ServeSubscription is the public /sub/:token endpoint (no auth required).
func (h *Handler) ServeSubscription(c *gin.Context) {
	token := c.Param("token")
	target := c.Query("target")
	if target == "" {
		target = exporter.DetectFormat(c.GetHeader("User-Agent"))
	}

	// 先查 Token 表（通过分享管理创建的）
	if tok, ok := h.store.GetTokenByValue(token); ok {
		if tok.Exp != nil && time.Now().Unix() > *tok.Exp {
			c.String(http.StatusGone, "subscription has expired")
			return
		}
		var nodes []store.Node
		if tok.Type == "sub" || tok.Type == "" {
			if sub, ok := h.store.GetSubscriptionByName(tok.Name); ok {
				nodes = sub.Nodes
			}
		} else if tok.Type == "col" {
			if col, ok := h.store.GetCollectionByName(tok.Name); ok {
				nodes = h.store.GetCollectionNodes(col)
			}
		}
		h.writeExport(c, target, nodes)
		return
	}

	// 再查 Collection token（旧机制）
	if col, ok := h.store.GetCollectionByToken(token); ok {
		if col.IsExpired() {
			c.String(http.StatusGone, "subscription has expired")
			return
		}
		nodes := h.store.GetCollectionNodes(col)
		h.writeExport(c, target, nodes)
		return
	}

	c.String(http.StatusNotFound, "subscription not found")
}

// ─── Utils ────────────────────────────────────────────────────────────────────

func (h *Handler) getEnv(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"backend": "Node",
			"version": "2.16.21",
			"feature": gin.H{
				"share": true,
			},
			"meta": gin.H{
				"node": gin.H{
					"env": gin.H{
						"SUB_STORE_BACKEND_CUSTOM_NAME":      "Sub-Store Go",
						"SUB_STORE_FRONTEND_BACKEND_PATH":    h.backendPath,
					},
				},
			},
		},
	})
}

func (h *Handler) refresh(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
}

// ─── Subscriptions ─────────────────────────────────────────────────────────────

// Sub-Store format: { status, data: Sub[] }
func (h *Handler) listSubs(c *gin.Context) {
	subs := h.store.GetSubscriptions()
	result := make([]map[string]interface{}, 0, len(subs))
	for _, s := range subs {
		result = append(result, subToSSFormat(s))
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *Handler) getOneSub(c *gin.Context) {
	name := decode(c.Param("name"))
	sub, ok := h.store.GetSubscriptionByName(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "failed", "error": gin.H{"message": "not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": subToSSFormat(*sub)})
}

func (h *Handler) createSub(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	sub := ssFormatToSub(body)
	sub.ID = store.NewID()
	sub.UpdatedAt = time.Now()
	sub.Enabled = true

	// Fetch nodes if remote
	if sub.SourceType == store.SourceURL && sub.URL != "" {
		nodes, err := fetchAndParse(sub.URL)
		if err == nil {
			assignIDs(nodes)
			sub.Nodes = nodes
		} else {
			sub.Nodes = []store.Node{}
		}
	} else if sub.SourceType == store.SourceLocal && sub.LocalContent != "" {
		nodes, _ := parser.ParseSubscription(sub.LocalContent)
		assignIDs(nodes)
		sub.Nodes = nodes
	} else {
		sub.Nodes = []store.Node{}
	}
	sub.NodeCount = len(sub.Nodes)

	if err := h.store.AddSubscription(sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": subToSSFormat(sub)})
}

func (h *Handler) editSub(c *gin.Context) {
	name := decode(c.Param("name"))
	sub, ok := h.store.GetSubscriptionByName(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "failed", "error": gin.H{"message": "not found"}})
		return
	}
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	updated := ssFormatToSub(body)
	// Merge fields
	if updated.Name != "" {
		sub.Name = updated.Name
	}
	if updated.URL != "" {
		sub.URL = updated.URL
		sub.SourceType = store.SourceURL
	}
	if updated.LocalContent != "" {
		sub.LocalContent = updated.LocalContent
		if sub.SourceType == store.SourceLocal || sub.SourceType == store.SourceFile {
			nodes, _ := parser.ParseSubscription(updated.LocalContent)
			assignIDs(nodes)
			sub.Nodes = nodes
			sub.NodeCount = len(nodes)
		}
	}
	if v, ok := body["source"]; ok {
		if src, _ := v.(string); src == "local" {
			sub.SourceType = store.SourceLocal
		} else if src == "remote" {
			sub.SourceType = store.SourceURL
		}
	}
	// Copy other ss-format fields into extra
	for _, k := range []string{"displayName", "display-name", "remark", "icon", "isIconColor", "ua", "tag", "process", "mergeSources", "subUserinfo"} {
		if v, ok := body[k]; ok {
			if sub.Extra == nil {
				sub.Extra = make(map[string]interface{})
			}
			sub.Extra[k] = v
		}
	}
	sub.UpdatedAt = time.Now()
	if err := h.store.UpdateSubscription(*sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": subToSSFormat(*sub)})
}

func (h *Handler) deleteSub(c *gin.Context) {
	name := decode(c.Param("name"))
	sub, ok := h.store.GetSubscriptionByName(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "failed", "error": gin.H{"message": "not found"}})
		return
	}
	if err := h.store.DeleteSubscription(sub.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
}

func (h *Handler) sortSubs(c *gin.Context) {
	// Accept array of names and reorder
	var names []string
	if err := c.ShouldBindJSON(&names); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	h.store.ReorderSubscriptions(names)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
}

// ─── Collections ──────────────────────────────────────────────────────────────

func (h *Handler) listCollections(c *gin.Context) {
	cols := h.store.GetCollections()
	result := make([]map[string]interface{}, 0, len(cols))
	for _, col := range cols {
		result = append(result, colToSSFormat(col, h.store))
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *Handler) getOneCollection(c *gin.Context) {
	name := decode(c.Param("name"))
	col, ok := h.store.GetCollectionByName(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "failed", "error": gin.H{"message": "not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": colToSSFormat(*col, h.store)})
}

func (h *Handler) createCollection(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	col := ssFormatToCollection(body)
	col.ID = store.NewID()
	col.Enabled = true
	col.UpdatedAt = time.Now()
	if err := h.store.AddCollection(col); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": colToSSFormat(col, h.store)})
}

func (h *Handler) editCollection(c *gin.Context) {
	name := decode(c.Param("name"))
	col, ok := h.store.GetCollectionByName(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "failed", "error": gin.H{"message": "not found"}})
		return
	}
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	updated := ssFormatToCollection(body)
	if updated.Name != "" {
		col.Name = updated.Name
	}
	if len(updated.SubIDs) > 0 {
		col.SubIDs = updated.SubIDs
	}
	// sync subscriptions field (original uses names, we store IDs)
	if subs, ok := body["subscriptions"]; ok {
		if subNames, ok := subs.([]interface{}); ok {
			var ids []string
			for _, n := range subNames {
				if name, ok := n.(string); ok {
					if sub, ok := h.store.GetSubscriptionByName(name); ok {
						ids = append(ids, sub.ID)
					}
				}
			}
			if len(ids) > 0 {
				col.SubIDs = ids
			}
		}
	}
	for _, k := range []string{"displayName", "display-name", "remark", "icon", "isIconColor", "tag", "process", "subscriptionTags"} {
		if v, ok := body[k]; ok {
			if col.Extra == nil {
				col.Extra = make(map[string]interface{})
			}
			col.Extra[k] = v
		}
	}
	col.UpdatedAt = time.Now()
	if err := h.store.UpdateCollection(*col); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": colToSSFormat(*col, h.store)})
}

func (h *Handler) deleteCollection(c *gin.Context) {
	name := decode(c.Param("name"))
	col, ok := h.store.GetCollectionByName(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "failed", "error": gin.H{"message": "not found"}})
		return
	}
	if err := h.store.DeleteCollection(col.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
}

func (h *Handler) sortCollections(c *gin.Context) {
	var names []string
	if err := c.ShouldBindJSON(&names); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	h.store.ReorderCollections(names)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
}

// ─── Tokens (Share) ──────────────────────────────────────────────────────────

func (h *Handler) listTokens(c *gin.Context) {
	typef := c.Query("type")
	name := c.Query("name")
	tokens := h.store.GetTokens(typef, name)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": tokens})
}

func (h *Handler) createToken(c *gin.Context) {
	var body struct {
		Payload struct {
			Type        string  `json:"type"`
			Name        string  `json:"name"`
			DisplayName string  `json:"displayName"`
			Remark      string  `json:"remark"`
			Token       string  `json:"token"`
		} `json:"payload"`
		Options struct {
			ExpiresIn interface{} `json:"expiresIn"`
		} `json:"options"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}

	tok := store.Token{
		ID:          store.NewID(),
		Type:        body.Payload.Type,
		Name:        body.Payload.Name,
		DisplayName: body.Payload.DisplayName,
		Remark:      body.Payload.Remark,
		CreatedAt:   time.Now().Unix(),
	}
	if body.Payload.Token != "" {
		tok.Token = body.Payload.Token
	} else {
		tok.Token = store.NewToken()
	}
	// parse expiresIn
	if body.Options.ExpiresIn != nil {
		expSecs := parseExpiresIn(body.Options.ExpiresIn)
		if expSecs > 0 {
			exp := time.Now().Unix() + expSecs
			tok.Exp = &exp
		}
	}

	if err := h.store.AddToken(tok); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "failed", "error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": tokenToMap(tok)})
}

func (h *Handler) deleteToken(c *gin.Context) {
	token := decode(c.Param("token"))
	h.store.DeleteToken(token)
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
}

func (h *Handler) sortTokens(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
}

// ─── Download ─────────────────────────────────────────────────────────────────

// DownloadCollectionSub handles /download/collection/:name
func (h *Handler) DownloadCollectionSub(c *gin.Context) {
	name := decode(c.Param("name"))
	col, ok := h.store.GetCollectionByName(name)
	if !ok {
		c.String(http.StatusNotFound, "not found")
		return
	}
	nodes := h.store.GetCollectionNodes(col)
	target := c.Query("target")
	if target == "" {
		target = exporter.DetectFormat(c.GetHeader("User-Agent"))
	}
	h.writeExport(c, target, nodes)
}

// DownloadSub is public (no auth) — clients use this directly.
func (h *Handler) DownloadSub(c *gin.Context) {
	name := decode(c.Param("name"))
	// Check if it's a collection first
	col, colOk := h.store.GetCollectionByName(name)
	var nodes []store.Node
	if colOk {
		nodes = h.store.GetCollectionNodes(col)
	} else if sub, subOk := h.store.GetSubscriptionByName(name); subOk {
		nodes = sub.Nodes
	} else {
		// Try token
		if tok, ok := h.store.GetTokenByValue(name); ok {
			if tok.Exp != nil && time.Now().Unix() > *tok.Exp {
				c.String(http.StatusGone, "token expired")
				return
			}
			if tok.Type == "sub" {
				if sub, ok := h.store.GetSubscriptionByName(tok.Name); ok {
					nodes = sub.Nodes
				}
			} else if tok.Type == "col" {
				if col, ok := h.store.GetCollectionByName(tok.Name); ok {
					nodes = h.store.GetCollectionNodes(col)
				}
			}
		} else {
			c.String(http.StatusNotFound, "not found")
			return
		}
	}

	target := c.Query("target")
	if target == "" {
		target = exporter.DetectFormat(c.GetHeader("User-Agent"))
	}
	h.writeExport(c, target, nodes)
}

// ─── Flow (stub) ──────────────────────────────────────────────────────────────

func (h *Handler) getFlow(c *gin.Context) {
	name := decode(c.Param("name"))
	sub, ok := h.store.GetSubscriptionByName(name)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status": "failed",
			"error":  gin.H{"code": "NO_FLOW_INFO", "type": "NO_FLOW_INFO", "message": "no flow info"},
		})
		return
	}
	// If URL is set, try fetching flow from remote
	if sub.URL != "" && sub.SourceType == store.SourceURL {
		client := &http.Client{Timeout: 8 * time.Second}
		req, _ := http.NewRequest("GET", sub.URL, nil)
		req.Header.Set("User-Agent", "ClashforWindows/0.20.39")
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			userinfo := resp.Header.Get("subscription-userinfo")
			if userinfo != "" {
				c.JSON(http.StatusOK, gin.H{
					"status": "success",
					"data":   parseUserinfo(userinfo, sub.URL),
				})
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"status": "noFlow",
		},
	})
}

// ─── Preview (stub) ───────────────────────────────────────────────────────────

func (h *Handler) previewSub(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": []interface{}{}})
}

// ─── Files (stub) ────────────────────────────────────────────────────────────

func (h *Handler) listFiles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": []interface{}{}})
}

// ─── Settings (stub) ─────────────────────────────────────────────────────────

func (h *Handler) getSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": map[string]interface{}{}})
}

func (h *Handler) patchSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": nil})
}

// ─── Artifacts (stub) ────────────────────────────────────────────────────────

func (h *Handler) listArtifacts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": []interface{}{}})
}

// ─── Node info (stub) ────────────────────────────────────────────────────────

func (h *Handler) nodeInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": []interface{}{}})
}

// ─── Export ───────────────────────────────────────────────────────────────────

func normalizeFormat(format string) string {
	// 原版 Sub-Store 前端传的是大驼峰格式，统一转成我们的小写格式
	switch format {
	case "Surge", "SurgeMac":
		return "surge"
	case "ClashMeta", "Clash", "Stash", "Surfboard", "Egern":
		return "clash"
	case "QX":
		return "qx"
	case "Loon":
		return "loon"
	case "ShadowRocket":
		return "shadowrocket"
	case "sing-box":
		return "singbox"
	case "V2Ray", "URI":
		return "base64"
	case "JSON":
		return "singbox"
	}
	return strings.ToLower(format)
}

func (h *Handler) writeExport(c *gin.Context, format string, nodes []store.Node) {
	format = normalizeFormat(format)
	switch format {
	case "clash":
		out, err := exporter.ToClash(nodes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Disposition", "attachment; filename=config.yaml")
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(out))
	case "surge":
		c.Header("Content-Disposition", "attachment; filename=surge.conf")
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(exporter.ToSurge(nodes)))
	case "qx":
		c.Header("Content-Disposition", "attachment; filename=quantumult-x.conf")
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(exporter.ToQuantumultX(nodes)))
	case "loon":
		c.Header("Content-Disposition", "attachment; filename=loon.conf")
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(exporter.ToLoon(nodes)))
	case "shadowrocket":
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(exporter.ToShadowrocket(nodes)))
	case "singbox":
		out, err := exporter.ToSingBox(nodes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Disposition", "attachment; filename=singbox.json")
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(out))
	default:
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(exporter.ToBase64(nodes)))
	}
}

// ─── Format converters ────────────────────────────────────────────────────────

// subToSSFormat converts our internal Subscription to Sub-Store's wire format
func subToSSFormat(s store.Subscription) map[string]interface{} {
	source := "remote"
	if s.SourceType == store.SourceLocal || s.SourceType == store.SourceFile {
		source = "local"
	}
	m := map[string]interface{}{
		"name":    s.Name,
		"url":     s.URL,
		"source":  source,
		"content": s.LocalContent,
		"process": []interface{}{},
		"tag":     []string{},
	}
	// Merge extra fields
	for k, v := range s.Extra {
		m[k] = v
	}
	return m
}

// ssFormatToSub converts Sub-Store wire format to our internal Subscription
func ssFormatToSub(body map[string]interface{}) store.Subscription {
	sub := store.Subscription{
		Params: make(map[string]string),
		Extra:  make(map[string]interface{}),
	}
	if v, ok := body["name"].(string); ok {
		sub.Name = v
	}
	if v, ok := body["url"].(string); ok {
		sub.URL = v
	}
	if v, ok := body["content"].(string); ok {
		sub.LocalContent = v
	}
	source, _ := body["source"].(string)
	if source == "local" {
		sub.SourceType = store.SourceLocal
	} else {
		sub.SourceType = store.SourceURL
	}
	// stash extra SS fields
	for _, k := range []string{"displayName", "display-name", "remark", "icon", "isIconColor", "ua", "tag", "process", "mergeSources", "subUserinfo"} {
		if v, ok := body[k]; ok {
			sub.Extra[k] = v
		}
	}
	return sub
}

// colToSSFormat converts our Collection to Sub-Store format
func colToSSFormat(col store.Collection, s *store.Store) map[string]interface{} {
	// Convert sub IDs → sub names
	var subNames []string
	for _, id := range col.SubIDs {
		if sub, ok := s.GetSubscription(id); ok {
			subNames = append(subNames, sub.Name)
		}
	}
	if subNames == nil {
		subNames = []string{}
	}
	m := map[string]interface{}{
		"name":          col.Name,
		"subscriptions": subNames,
		"process":       []interface{}{},
		"tag":           []string{},
	}
	for k, v := range col.Extra {
		m[k] = v
	}
	return m
}

// ssFormatToCollection converts Sub-Store collection format to our internal Collection
func ssFormatToCollection(body map[string]interface{}) store.Collection {
	col := store.Collection{
		Extra: make(map[string]interface{}),
	}
	if v, ok := body["name"].(string); ok {
		col.Name = v
	}
	// subscriptions field = array of sub names; resolve to IDs later
	if v, ok := body["subscriptions"].([]interface{}); ok {
		for _, n := range v {
			if name, ok := n.(string); ok {
				col.SubIDs = append(col.SubIDs, name) // temp: store names, resolve in caller
			}
		}
	}
	for _, k := range []string{"displayName", "display-name", "remark", "icon", "isIconColor", "tag", "process", "subscriptionTags"} {
		if v, ok := body[k]; ok {
			col.Extra[k] = v
		}
	}
	return col
}

func tokenToMap(t store.Token) map[string]interface{} {
	m := map[string]interface{}{
		"type":        t.Type,
		"name":        t.Name,
		"displayName": t.DisplayName,
		"remark":      t.Remark,
		"token":       t.Token,
		"createdAt":   t.CreatedAt * 1000, // 转毫秒，dayjs 期望毫秒
	}
	if t.Exp != nil {
		m["exp"] = *t.Exp * 1000 // 转毫秒
	}
	return m
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func fetchAndParse(rawURL string) ([]store.Node, error) {
	if isDirectURI(rawURL) {
		node, err := parser.ParseURI(rawURL)
		if err != nil {
			return nil, err
		}
		return []store.Node{node}, nil
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parser.ParseSubscription(string(body))
}

func isDirectURI(s string) bool {
	for _, pfx := range []string{"ss://", "vmess://", "trojan://", "vless://", "hy2://", "hysteria2://"} {
		if strings.HasPrefix(s, pfx) {
			return true
		}
	}
	return false
}

func assignIDs(nodes []store.Node) {
	for i := range nodes {
		nodes[i].ID = fmt.Sprintf("%s-%d", store.NewID(), i)
	}
}

func decode(s string) string {
	// gin already URL-decodes path params, just trim
	return strings.TrimSpace(s)
}

func parseExpiresIn(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case string:
		// e.g. "7d", "30d", "1m", "1y"
		if len(val) < 2 {
			return 0
		}
		unit := val[len(val)-1]
		var num int64
		fmt.Sscanf(val[:len(val)-1], "%d", &num)
		switch unit {
		case 'd':
			return num * 86400
		case 'm':
			return num * 86400 * 30
		case 'y':
			return num * 86400 * 365
		case 's':
			return num
		}
	}
	return 0
}

func parseUserinfo(userinfo, url string) map[string]interface{} {
	m := map[string]interface{}{"status": "success"}
	parts := strings.Split(userinfo, ";")
	var upload, download, total int64
	var expire int64
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch k {
		case "upload":
			fmt.Sscanf(v, "%d", &upload)
		case "download":
			fmt.Sscanf(v, "%d", &download)
		case "total":
			fmt.Sscanf(v, "%d", &total)
		case "expire":
			fmt.Sscanf(v, "%d", &expire)
		}
	}
	usage := map[string]interface{}{"upload": upload, "download": download}
	data := map[string]interface{}{
		"total": total,
		"usage": usage,
	}
	if expire > 0 {
		data["expires"] = expire
		remaining := time.Unix(expire, 0).Sub(time.Now())
		if remaining > 0 {
			data["remainingDays"] = int(remaining.Hours() / 24)
		}
	}
	m["data"] = data
	return m
}
