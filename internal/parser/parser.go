package parser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/evecus/sub/internal/store"
)

// ParseSubscription detects format and parses nodes
func ParseSubscription(content string) ([]store.Node, error) {
	content = strings.TrimSpace(content)

	// Try Clash YAML (contains "proxies:")
	if strings.Contains(content, "proxies:") {
		return parseClashYAML(content)
	}

	// Try Base64 decode
	decoded, err := base64Decode(content)
	if err == nil && looksLikeNodeList(decoded) {
		content = decoded
	}

	// Parse line by line
	var nodes []store.Node
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		node, err := ParseURI(line)
		if err == nil {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

func looksLikeNodeList(s string) bool {
	for _, pfx := range []string{"ss://", "vmess://", "trojan://", "vless://", "hy2://", "hysteria2://", "proxies:"} {
		if strings.Contains(s, pfx) {
			return true
		}
	}
	return false
}

// ParseURI parses a single proxy URI
func ParseURI(uri string) (store.Node, error) {
	switch {
	case strings.HasPrefix(uri, "ss://"):
		return parseSS(uri)
	case strings.HasPrefix(uri, "vmess://"):
		return parseVMess(uri)
	case strings.HasPrefix(uri, "trojan://"):
		return parseTrojan(uri)
	case strings.HasPrefix(uri, "vless://"):
		return parseVLESS(uri)
	case strings.HasPrefix(uri, "hy2://"), strings.HasPrefix(uri, "hysteria2://"):
		return parseHysteria2(uri)
	default:
		return store.Node{}, fmt.Errorf("unknown protocol")
	}
}

func parseSS(uri string) (store.Node, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return store.Node{}, err
	}
	name, _ := url.QueryUnescape(u.Fragment)
	node := store.Node{
		Type: store.NodeSS, Name: name, RawURI: uri,
		Params: make(map[string]string), Country: detectCountry(name),
	}
	if u.User != nil {
		node.Server = u.Hostname()
		port, _ := strconv.Atoi(u.Port())
		node.Port = port
		node.Params["method"] = u.User.Username()
		node.Params["password"], _ = u.User.Password()
	} else {
		raw := strings.TrimPrefix(uri, "ss://")
		if idx := strings.Index(raw, "#"); idx != -1 {
			raw = raw[:idx]
		}
		decoded, err := base64Decode(raw)
		if err != nil {
			return store.Node{}, err
		}
		atIdx := strings.LastIndex(decoded, "@")
		if atIdx == -1 {
			return store.Node{}, fmt.Errorf("invalid ss uri")
		}
		parts := strings.SplitN(decoded[:atIdx], ":", 2)
		if len(parts) == 2 {
			node.Params["method"] = parts[0]
			node.Params["password"] = parts[1]
		}
		hostPort := decoded[atIdx+1:]
		if hp := strings.LastIndex(hostPort, ":"); hp != -1 {
			node.Server = hostPort[:hp]
			node.Port, _ = strconv.Atoi(hostPort[hp+1:])
		}
	}
	if node.Name == "" {
		node.Name = fmt.Sprintf("%s:%d", node.Server, node.Port)
	}
	return node, nil
}

func parseVMess(uri string) (store.Node, error) {
	raw := strings.TrimPrefix(uri, "vmess://")
	decoded, err := base64Decode(raw)
	if err != nil {
		return store.Node{}, err
	}
	var v map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &v); err != nil {
		return store.Node{}, err
	}
	node := store.Node{Type: store.NodeVMess, RawURI: uri, Params: make(map[string]string)}
	if name, ok := v["ps"].(string); ok {
		node.Name = name
	}
	if add, ok := v["add"].(string); ok {
		node.Server = add
	}
	if port, ok := v["port"]; ok {
		switch p := port.(type) {
		case float64:
			node.Port = int(p)
		case string:
			node.Port, _ = strconv.Atoi(p)
		}
	}
	for _, key := range []string{"id", "aid", "net", "type", "host", "path", "tls", "sni"} {
		if val, ok := v[key].(string); ok {
			node.Params[key] = val
		}
	}
	node.Country = detectCountry(node.Name)
	if node.Name == "" {
		node.Name = fmt.Sprintf("%s:%d", node.Server, node.Port)
	}
	return node, nil
}

func parseTrojan(uri string) (store.Node, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return store.Node{}, err
	}
	name, _ := url.QueryUnescape(u.Fragment)
	port, _ := strconv.Atoi(u.Port())
	node := store.Node{
		Type: store.NodeTrojan, Name: name, Server: u.Hostname(), Port: port,
		RawURI: uri, Params: make(map[string]string), Country: detectCountry(name),
	}
	node.Params["password"] = u.User.Username()
	for k, v := range u.Query() {
		if len(v) > 0 {
			node.Params[k] = v[0]
		}
	}
	if node.Name == "" {
		node.Name = fmt.Sprintf("%s:%d", node.Server, node.Port)
	}
	return node, nil
}

func parseVLESS(uri string) (store.Node, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return store.Node{}, err
	}
	name, _ := url.QueryUnescape(u.Fragment)
	port, _ := strconv.Atoi(u.Port())
	node := store.Node{
		Type: store.NodeVLESS, Name: name, Server: u.Hostname(), Port: port,
		RawURI: uri, Params: make(map[string]string), Country: detectCountry(name),
	}
	node.Params["uuid"] = u.User.Username()
	for k, v := range u.Query() {
		if len(v) > 0 {
			node.Params[k] = v[0]
		}
	}
	if node.Name == "" {
		node.Name = fmt.Sprintf("%s:%d", node.Server, node.Port)
	}
	return node, nil
}

func parseHysteria2(uri string) (store.Node, error) {
	raw := uri
	if strings.HasPrefix(raw, "hysteria2://") {
		raw = "hy2://" + strings.TrimPrefix(raw, "hysteria2://")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return store.Node{}, err
	}
	name, _ := url.QueryUnescape(u.Fragment)
	port, _ := strconv.Atoi(u.Port())
	node := store.Node{
		Type: store.NodeHysteria, Name: name, Server: u.Hostname(), Port: port,
		RawURI: uri, Params: make(map[string]string), Country: detectCountry(name),
	}
	node.Params["auth"] = u.User.Username()
	for k, v := range u.Query() {
		if len(v) > 0 {
			node.Params[k] = v[0]
		}
	}
	if node.Name == "" {
		node.Name = fmt.Sprintf("%s:%d", node.Server, node.Port)
	}
	return node, nil
}

// ---- Clash YAML parser (手写，不依赖yaml库) ----

func parseClashYAML(content string) ([]store.Node, error) {
	var nodes []store.Node
	inProxies := false
	var currentProxy map[string]string

	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		stripped := strings.TrimSpace(line)

		// Detect proxies: section
		if stripped == "proxies:" {
			inProxies = true
			continue
		}
		// New top-level section ends proxies
		if inProxies && len(line) > 0 && line[0] != ' ' && line[0] != '-' && line[0] != '#' {
			if currentProxy != nil {
				if n, ok := clashProxyToNode(currentProxy); ok {
					nodes = append(nodes, n)
				}
				currentProxy = nil
			}
			inProxies = false
			continue
		}

		if !inProxies {
			continue
		}
		if stripped == "" || stripped == "[]" {
			continue
		}

		// New proxy item
		if strings.HasPrefix(stripped, "- ") || stripped == "-" {
			if currentProxy != nil {
				if n, ok := clashProxyToNode(currentProxy); ok {
					nodes = append(nodes, n)
				}
			}
			currentProxy = make(map[string]string)
			rest := strings.TrimPrefix(stripped, "- ")
			if rest != "" && rest != "-" {
				parseYAMLKV(rest, currentProxy)
			}
			continue
		}

		// Continuation of current proxy
		if currentProxy != nil && strings.HasPrefix(line, "  ") {
			parseYAMLKV(stripped, currentProxy)
		}
	}

	// Don't forget last proxy
	if currentProxy != nil {
		if n, ok := clashProxyToNode(currentProxy); ok {
			nodes = append(nodes, n)
		}
	}

	return nodes, nil
}

func parseYAMLKV(line string, m map[string]string) {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return
	}
	key := strings.TrimSpace(line[:idx])
	val := strings.TrimSpace(line[idx+1:])
	// Remove surrounding quotes
	if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
		val = val[1 : len(val)-1]
	}
	m[key] = val
}

func clashProxyToNode(m map[string]string) (store.Node, bool) {
	name := m["name"]
	server := m["server"]
	portStr := m["port"]
	typeName := m["type"]

	if name == "" || server == "" || portStr == "" || typeName == "" {
		return store.Node{}, false
	}

	port, _ := strconv.Atoi(portStr)
	node := store.Node{
		Name:    name,
		Server:  server,
		Port:    port,
		Country: detectCountry(name),
		Params:  make(map[string]string),
	}

	switch strings.ToLower(typeName) {
	case "ss", "shadowsocks":
		node.Type = store.NodeSS
		node.Params["method"] = m["cipher"]
		node.Params["password"] = m["password"]
	case "vmess":
		node.Type = store.NodeVMess
		node.Params["id"] = m["uuid"]
		if net, ok := m["network"]; ok {
			node.Params["net"] = net
		}
		if m["tls"] == "true" {
			node.Params["tls"] = "tls"
		}
	case "trojan":
		node.Type = store.NodeTrojan
		node.Params["password"] = m["password"]
		if sni, ok := m["sni"]; ok {
			node.Params["sni"] = sni
		}
	case "vless":
		node.Type = store.NodeVLESS
		node.Params["uuid"] = m["uuid"]
		if flow, ok := m["flow"]; ok {
			node.Params["flow"] = flow
		}
		if m["tls"] == "true" {
			node.Params["security"] = "tls"
		}
	case "hysteria2":
		node.Type = store.NodeHysteria
		node.Params["auth"] = m["password"]
		if sni, ok := m["sni"]; ok {
			node.Params["sni"] = sni
		}
	default:
		node.Type = store.NodeUnknown
	}
	return node, true
}

// ---- helpers ----

func base64Decode(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	b, err := base64.StdEncoding.DecodeString(s)
	return string(b), err
}

func detectCountry(name string) string {
	upper := strings.ToUpper(name)
	flags := map[string]string{
		"🇭🇰": "HK", "🇺🇸": "US", "🇯🇵": "JP", "🇸🇬": "SG",
		"🇹🇼": "TW", "🇩🇪": "DE", "🇬🇧": "UK", "🇫🇷": "FR",
		"🇰🇷": "KR", "🇦🇺": "AU", "🇨🇦": "CA", "🇳🇱": "NL",
	}
	for flag, code := range flags {
		if strings.Contains(name, flag) {
			return code
		}
	}
	keywords := map[string]string{
		"HK": "HK", "HONGKONG": "HK", "香港": "HK",
		"US": "US", "USA": "US", "UNITED STATES": "US", "美国": "US", "美國": "US",
		"JP": "JP", "JAPAN": "JP", "日本": "JP",
		"SG": "SG", "SINGAPORE": "SG", "新加坡": "SG",
		"TW": "TW", "TAIWAN": "TW", "台湾": "TW", "台灣": "TW",
		"DE": "DE", "GERMANY": "DE", "德国": "DE",
		"UK": "UK", "UNITED KINGDOM": "UK", "英国": "UK",
		"KR": "KR", "KOREA": "KR", "韩国": "KR",
		"AU": "AU", "AUSTRALIA": "AU", "澳大利亚": "AU",
	}
	for kw, code := range keywords {
		if strings.Contains(upper, kw) {
			return code
		}
	}
	return ""
}
