# Node Manager

类 Sub-Store 的代理节点管理工具。单二进制运行，无需数据库。

## 功能

- **订阅管理**：添加、删除、刷新订阅，支持启用/禁用
- **节点解析**：支持 SS / VMess / Trojan / VLESS / Hysteria2
- **订阅格式**：支持 Base64 订阅和 Clash YAML 订阅
- **节点列表**：按地区分组展示，支持搜索和协议过滤，点击复制节点链接
- **多格式导出**：Clash Meta YAML / Surge .conf / Base64，可勾选订阅合并导出

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+

### 构建

```bash
# 克隆项目
git clone https://github.com/yourname/node-manager
cd node-manager

# 一键构建（安装依赖 → 构建前端 → 编译二进制）
make build

# 运行
./node-manager
```

打开浏览器访问 `http://localhost:8080`

### 自定义端口

```bash
PORT=9090 ./node-manager
```

### 跨平台编译

```bash
# 编译全平台版本到 ./dist/
make release
```

## 开发模式

```bash
# 终端 1：启动 Go 后端
make dev-go

# 终端 2：启动前端开发服务器（含热重载，代理到 :8080）
make dev-web
```

访问 `http://localhost:5173`

## 数据存储

数据存储在 `~/.node-manager/data.json`，包含所有订阅和节点信息。

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/subscriptions` | 获取订阅列表 |
| POST | `/api/subscriptions` | 添加订阅 |
| DELETE | `/api/subscriptions/:id` | 删除订阅 |
| POST | `/api/subscriptions/:id/refresh` | 刷新节点 |
| PUT | `/api/subscriptions/:id/toggle` | 启用/禁用订阅 |
| GET | `/api/nodes?sub_id=xxx` | 获取节点（可多个 sub_id） |
| GET | `/api/export/clash?sub_id=xxx` | 导出 Clash 配置 |
| GET | `/api/export/surge?sub_id=xxx` | 导出 Surge 配置 |
| GET | `/api/export/base64?sub_id=xxx` | 导出 Base64 订阅 |

## 支持的协议

| 协议 | URI 前缀 | 说明 |
|------|----------|------|
| Shadowsocks | `ss://` | 标准格式和 Legacy Base64 |
| VMess | `vmess://` | V2Ray 标准 JSON |
| Trojan | `trojan://` | 标准 URI |
| VLESS | `vless://` | Xray 标准 URI |
| Hysteria2 | `hy2://` / `hysteria2://` | 新版 Hysteria |

订阅格式支持：**Base64 多行节点**、**Clash YAML**（含 `proxies:` 字段）
