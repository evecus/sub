package exporter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/evecus/sub/internal/store"
)

// ---- Clash Meta (手写YAML，不依赖yaml.v3) ----

func ToClash(nodes []store.Node) (string, error) {
	var sb strings.Builder

	sb.WriteString("port: 7890\n")
	sb.WriteString("socks-port: 7891\n")
	sb.WriteString("allow-lan: false\n")
	sb.WriteString("mode: rule\n")
	sb.WriteString("log-level: info\n")
	sb.WriteString("external-controller: 127.0.0.1:9090\n\n")
	sb.WriteString("proxies:\n")

	var names []string
	for _, n := range nodes {
		block := nodeToClashYAML(n)
		if block == "" {
			continue
		}
		sb.WriteString(block)
		names = append(names, n.Name)
	}

	if len(names) == 0 {
		sb.WriteString("[]\n")
	}

	sb.WriteString("\nproxy-groups:\n")
	sb.WriteString("  - name: \"🚀 节点选择\"\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	for _, name := range names {
		sb.WriteString(fmt.Sprintf("      - %s\n", yamlQuote(name)))
	}

	if len(names) > 0 {
		sb.WriteString("  - name: \"🔄 自动选择\"\n")
		sb.WriteString("    type: url-test\n")
		sb.WriteString(fmt.Sprintf("    url: %q\n", "http://www.gstatic.com/generate_204"))
		sb.WriteString("    interval: 300\n")
		sb.WriteString("    proxies:\n")
		for _, name := range names {
			sb.WriteString(fmt.Sprintf("      - %s\n", yamlQuote(name)))
		}
	}

	sb.WriteString("\nrules:\n")
	sb.WriteString("  - GEOIP,CN,DIRECT\n")
	sb.WriteString("  - MATCH,🚀 节点选择\n")

	return sb.String(), nil
}

func nodeToClashYAML(n store.Node) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  - name: %s\n", yamlQuote(n.Name)))
	sb.WriteString(fmt.Sprintf("    server: %s\n", n.Server))
	sb.WriteString(fmt.Sprintf("    port: %d\n", n.Port))

	switch n.Type {
	case store.NodeSS:
		sb.WriteString("    type: ss\n")
		sb.WriteString(fmt.Sprintf("    cipher: %s\n", n.Params["method"]))
		sb.WriteString(fmt.Sprintf("    password: %s\n", yamlQuote(n.Params["password"])))
	case store.NodeVMess:
		sb.WriteString("    type: vmess\n")
		sb.WriteString(fmt.Sprintf("    uuid: %s\n", n.Params["id"]))
		sb.WriteString("    alterId: 0\n")
		sb.WriteString("    cipher: auto\n")
		if net, ok := n.Params["net"]; ok && net != "" {
			sb.WriteString(fmt.Sprintf("    network: %s\n", net))
		}
		if n.Params["tls"] == "tls" {
			sb.WriteString("    tls: true\n")
		}
	case store.NodeTrojan:
		sb.WriteString("    type: trojan\n")
		sb.WriteString(fmt.Sprintf("    password: %s\n", yamlQuote(n.Params["password"])))
		if sni := n.Params["sni"]; sni != "" {
			sb.WriteString(fmt.Sprintf("    sni: %s\n", sni))
		}
	case store.NodeVLESS:
		sb.WriteString("    type: vless\n")
		sb.WriteString(fmt.Sprintf("    uuid: %s\n", n.Params["uuid"]))
		if flow := n.Params["flow"]; flow != "" {
			sb.WriteString(fmt.Sprintf("    flow: %s\n", flow))
		}
		if sec := n.Params["security"]; sec == "tls" || sec == "reality" {
			sb.WriteString("    tls: true\n")
		}
	case store.NodeHysteria:
		sb.WriteString("    type: hysteria2\n")
		sb.WriteString(fmt.Sprintf("    password: %s\n", yamlQuote(n.Params["auth"])))
		if sni := n.Params["sni"]; sni != "" {
			sb.WriteString(fmt.Sprintf("    sni: %s\n", sni))
		}
	default:
		return ""
	}
	return sb.String()
}

func yamlQuote(s string) string {
	if strings.ContainsAny(s, `:{}[]|>&*!,'"%@` + "`\n\t") || strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		escaped := strings.ReplaceAll(s, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}

// ---- Surge ----

func ToSurge(nodes []store.Node) string {
	var sb strings.Builder
	sb.WriteString("[Proxy]\n")
	var names []string
	for _, n := range nodes {
		line := nodeToSurgeLine(n)
		if line != "" {
			sb.WriteString(line + "\n")
			names = append(names, n.Name)
		}
	}
	sb.WriteString("\n[Proxy Group]\n🚀 Proxy = select, DIRECT")
	for _, name := range names {
		sb.WriteString(", " + name)
	}
	sb.WriteString("\n\n[Rule]\nGEOIP,CN,DIRECT\nFINAL,🚀 Proxy\n")
	return sb.String()
}

func nodeToSurgeLine(n store.Node) string {
	safe := strings.ReplaceAll(n.Name, ",", "_")
	switch n.Type {
	case store.NodeSS:
		return fmt.Sprintf("%s = ss, %s, %d, encrypt-method=%s, password=%s",
			safe, n.Server, n.Port, n.Params["method"], n.Params["password"])
	case store.NodeVMess:
		line := fmt.Sprintf("%s = vmess, %s, %d, username=%s", safe, n.Server, n.Port, n.Params["id"])
		if n.Params["tls"] == "tls" {
			line += ", tls=true"
		}
		if n.Params["net"] == "ws" {
			line += ", ws=true"
			if path := n.Params["path"]; path != "" {
				line += ", ws-path=" + path
			}
		}
		return line
	case store.NodeTrojan:
		line := fmt.Sprintf("%s = trojan, %s, %d, password=%s", safe, n.Server, n.Port, n.Params["password"])
		if sni := n.Params["sni"]; sni != "" {
			line += ", sni=" + sni
		}
		return line
	case store.NodeVLESS:
		line := fmt.Sprintf("%s = vless, %s, %d, username=%s", safe, n.Server, n.Port, n.Params["uuid"])
		if n.Params["security"] == "tls" {
			line += ", tls=true"
		}
		return line
	case store.NodeHysteria:
		return fmt.Sprintf("%s = hysteria2, %s, %d, password=%s", safe, n.Server, n.Port, n.Params["auth"])
	default:
		return ""
	}
}

// ---- Quantumult X ----

func ToQuantumultX(nodes []store.Node) string {
	var sb strings.Builder
	sb.WriteString("[server_local]\n")
	for _, n := range nodes {
		line := nodeToQXLine(n)
		if line != "" {
			sb.WriteString(line + "\n")
		}
	}
	return sb.String()
}

func nodeToQXLine(n store.Node) string {
	tag := n.Name
	switch n.Type {
	case store.NodeSS:
		return fmt.Sprintf("shadowsocks=%s:%d, method=%s, password=%s, tag=%s",
			n.Server, n.Port, n.Params["method"], n.Params["password"], tag)
	case store.NodeVMess:
		obfs := "none"
		if n.Params["net"] == "ws" {
			if n.Params["tls"] == "tls" {
				obfs = "wss"
			} else {
				obfs = "ws"
			}
		} else if n.Params["tls"] == "tls" {
			obfs = "over-tls"
		}
		line := fmt.Sprintf("vmess=%s:%d, method=chacha20-poly1305, password=%s, obfs=%s, tag=%s",
			n.Server, n.Port, n.Params["id"], obfs, tag)
		if path := n.Params["path"]; path != "" {
			line += ", obfs-uri=" + path
		}
		if host := n.Params["host"]; host != "" {
			line += ", obfs-host=" + host
		}
		return line
	case store.NodeTrojan:
		line := fmt.Sprintf("trojan=%s:%d, password=%s, over-tls=true, tag=%s",
			n.Server, n.Port, n.Params["password"], tag)
		if sni := n.Params["sni"]; sni != "" {
			line += ", tls-host=" + sni
		}
		return line
	case store.NodeVLESS:
		line := fmt.Sprintf("vless=%s:%d, password=%s, tag=%s",
			n.Server, n.Port, n.Params["uuid"], tag)
		if n.Params["security"] == "tls" {
			line += ", over-tls=true"
		}
		return line
	default:
		return ""
	}
}

// ---- Loon ----

func ToLoon(nodes []store.Node) string {
	var sb strings.Builder
	sb.WriteString("[Proxy]\n")
	var names []string
	for _, n := range nodes {
		line := nodeToLoonLine(n)
		if line != "" {
			sb.WriteString(line + "\n")
			names = append(names, n.Name)
		}
	}
	sb.WriteString("\n[Proxy Group]\nProxy = select,DIRECT")
	for _, name := range names {
		sb.WriteString("," + name)
	}
	sb.WriteString("\n\n[Rule]\nGEOIP,cn,DIRECT\nFINAL,Proxy\n")
	return sb.String()
}

func nodeToLoonLine(n store.Node) string {
	switch n.Type {
	case store.NodeSS:
		return fmt.Sprintf("%s = Shadowsocks,%s,%d,%s,\"%s\"",
			n.Name, n.Server, n.Port, n.Params["method"], n.Params["password"])
	case store.NodeVMess:
		transport := "tcp"
		if net := n.Params["net"]; net != "" {
			transport = net
		}
		tls := "false"
		if n.Params["tls"] == "tls" {
			tls = "true"
		}
		return fmt.Sprintf("%s = vmess,%s,%d,auto,\"%s\",transport=%s,tls=%s",
			n.Name, n.Server, n.Port, n.Params["id"], transport, tls)
	case store.NodeTrojan:
		line := fmt.Sprintf("%s = Trojan,%s,%d,\"%s\"", n.Name, n.Server, n.Port, n.Params["password"])
		if sni := n.Params["sni"]; sni != "" {
			line += ",sni=" + sni
		}
		return line
	case store.NodeVLESS:
		tls := "false"
		if n.Params["security"] == "tls" {
			tls = "true"
		}
		return fmt.Sprintf("%s = vless,%s,%d,\"%s\",tls=%s",
			n.Name, n.Server, n.Port, n.Params["uuid"], tls)
	case store.NodeHysteria:
		return fmt.Sprintf("%s = Hysteria2,%s,%d,\"%s\"", n.Name, n.Server, n.Port, n.Params["auth"])
	default:
		return ""
	}
}

// ---- Shadowrocket ----

func ToShadowrocket(nodes []store.Node) string {
	return ToBase64(nodes)
}

// ---- sing-box ----

func ToSingBox(nodes []store.Node) (string, error) {
	var outbounds []map[string]interface{}
	var tags []string

	for _, n := range nodes {
		ob := nodeToSingBoxOutbound(n)
		if ob == nil {
			continue
		}
		outbounds = append(outbounds, ob)
		tags = append(tags, n.Name)
	}

	selectorProxies := append([]string{"auto"}, tags...)
	header := []map[string]interface{}{
		{"tag": "proxy", "type": "selector", "outbounds": selectorProxies},
		{"tag": "auto", "type": "urltest", "outbounds": tags,
			"url": "https://www.gstatic.com/generate_204", "interval": "5m0s"},
		{"tag": "direct", "type": "direct"},
		{"tag": "block", "type": "block"},
	}
	allOutbounds := append(header, outbounds...)

	cfg := map[string]interface{}{
		"log": map[string]interface{}{"level": "info", "timestamp": true},
		"dns": map[string]interface{}{
			"servers": []map[string]interface{}{
				{"tag": "remote", "address": "https://8.8.8.8/dns-query", "detour": "proxy"},
				{"tag": "local", "address": "https://223.5.5.5/dns-query", "detour": "direct"},
			},
			"rules":  []map[string]interface{}{{"geosite": []string{"cn"}, "server": "local"}},
			"final":  "remote",
		},
		"inbounds": []map[string]interface{}{
			{"tag": "mixed", "type": "mixed", "listen": "127.0.0.1", "listen_port": 7890, "sniff": true},
		},
		"outbounds": allOutbounds,
		"route": map[string]interface{}{
			"rules":                   []map[string]interface{}{{"geoip": []string{"cn"}, "outbound": "direct"}, {"geosite": []string{"cn"}, "outbound": "direct"}},
			"final":                   "proxy",
			"auto_detect_interface":   true,
		},
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func nodeToSingBoxOutbound(n store.Node) map[string]interface{} {
	base := map[string]interface{}{
		"tag": n.Name, "server": n.Server, "server_port": n.Port,
	}
	switch n.Type {
	case store.NodeSS:
		base["type"] = "shadowsocks"
		base["method"] = n.Params["method"]
		base["password"] = n.Params["password"]
	case store.NodeVMess:
		base["type"] = "vmess"
		base["uuid"] = n.Params["id"]
		base["alter_id"] = 0
		base["security"] = "auto"
		if n.Params["tls"] == "tls" {
			base["tls"] = map[string]interface{}{"enabled": true}
		}
		if n.Params["net"] == "ws" {
			wsOpts := map[string]interface{}{}
			if path := n.Params["path"]; path != "" {
				wsOpts["path"] = path
			}
			if host := n.Params["host"]; host != "" {
				wsOpts["headers"] = map[string]string{"Host": host}
			}
			base["transport"] = map[string]interface{}{"type": "ws", "options": wsOpts}
		}
	case store.NodeTrojan:
		base["type"] = "trojan"
		base["password"] = n.Params["password"]
		tlsCfg := map[string]interface{}{"enabled": true}
		if sni := n.Params["sni"]; sni != "" {
			tlsCfg["server_name"] = sni
		}
		base["tls"] = tlsCfg
	case store.NodeVLESS:
		base["type"] = "vless"
		base["uuid"] = n.Params["uuid"]
		if flow := n.Params["flow"]; flow != "" {
			base["flow"] = flow
		}
		if sec := n.Params["security"]; sec == "tls" || sec == "reality" {
			tlsCfg := map[string]interface{}{"enabled": true}
			if sni := n.Params["sni"]; sni != "" {
				tlsCfg["server_name"] = sni
			}
			base["tls"] = tlsCfg
		}
	case store.NodeHysteria:
		base["type"] = "hysteria2"
		base["password"] = n.Params["auth"]
		tlsCfg := map[string]interface{}{"enabled": true}
		if sni := n.Params["sni"]; sni != "" {
			tlsCfg["server_name"] = sni
		}
		base["tls"] = tlsCfg
	default:
		return nil
	}
	return base
}

// ---- Base64 ----

func ToBase64(nodes []store.Node) string {
	var lines []string
	for _, n := range nodes {
		if n.RawURI != "" {
			lines = append(lines, n.RawURI)
		}
	}
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))
}

// DetectFormat guesses export format from User-Agent
func DetectFormat(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "clash"):
		return "clash"
	case strings.Contains(ua, "surge"):
		return "surge"
	case strings.Contains(ua, "quantumult"):
		return "qx"
	case strings.Contains(ua, "loon"):
		return "loon"
	case strings.Contains(ua, "shadowrocket"):
		return "shadowrocket"
	case strings.Contains(ua, "sing-box"), strings.Contains(ua, "singbox"):
		return "singbox"
	default:
		return "base64"
	}
}
