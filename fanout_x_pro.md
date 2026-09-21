# 🚀 Fanout-X Pro

**Fanout-X Pro** 是一款高性能、全功能、结合了 `3X-UI` 易用性与 `fanout` 自动化节点提取分发优势的现代化代理节点整合系统。

它可以帮助您从 3X-UI 面板、VPNGate 公共池、SOCKS5/HTTP 代理池以及各类订阅链接中自动采集、测速筛选，并生成结构化的 SOCKS5/HTTP 代理列表与 VLESS/VMess 订阅。

---

## 🌟 核心特性

- ⚡ **高性能 Go 架构**：单二进制文件运行，极低内存与 CPU 占用。
- 🔍 **多源节点采集引擎**：
  - 3X-UI 面板 API 自动同步（获取 Inbound/Clients）。
  - VPNGate 免费公共代理池获取。
  - SOCKS5 / HTTP 住宅代理池轮替与健康检查。
  - V2Ray / Clash 订阅链接解析。
- 📊 **智能测速与过滤**：并发 RTT 延迟测试，自动踢出失效或高延迟节点。
- 📋 **多格式一键提取**：
  - **SOCKS5/HTTP 格式**：`IP:Port` 或 `IP:Port:User:Pass`
  - **通用订阅**：VLESS / VMess / Trojan / ShadowSocks 链接与 Base64 订阅
  - **API 接口**：标准 JSON 数据流，供三方爬虫或脚本直接调用
- 🎨 **现代化 3X-UI 风格面板**：可视化节点列表、测速控制、实时日志与提取配置。
- 🛠️ **一键 VPS 安装**：集成 systemd 托管与自动更新。

---

## 📦 一键安装 (VPS)

只需在 Linux (Ubuntu/Debian/CentOS) VPS 上运行以下单条 Shell 命令：

```bash
bash <(curl -sSL https://raw.githubusercontent.com/YOUR_GITHUB_USERNAME/fanout-x-pro/main/install.sh)
```

---

## 🛠️ Docker 部署方式

如果你偏好使用 Docker 部署：

```bash
docker run -d \
  --name fanout-x-pro \
  --restart always \
  -p 8080:8080 \
  -v ./config.json:/app/config.json \
  yourdockerhub/fanout-x-pro:latest
```

---

## 📖 API 接口手册

### 1. 提取 SOCKS5 / HTTP 代理列表
- **URL**: `GET /api/v1/extract/proxies`
- **Query 参数**:
  - `type`: `socks5` | `http` (默认 `socks5`)
  - `format`: `txt` | `json` (默认 `txt`)
  - `max_delay`: 最大可用延迟毫秒数 (如 `500`)
- **示例**:
  ```bash
  curl "http://YOUR_VPS_IP:8080/api/v1/extract/proxies?type=socks5&format=txt&max_delay=300"
  ```
- **输出格式**:
  ```text
  192.168.1.100:1080:admin:pass123
  192.168.1.101:1080:admin:pass123
  ```

### 2. 获取节点通用订阅链接 (VLESS/VMess)
- **URL**: `GET /api/v1/extract/sub`
- **Query 参数**:
  - `format`: `raw` | `base64`
- **示例**:
  ```bash
  curl "http://YOUR_VPS_IP:8080/api/v1/extract/sub?format=base64"
  ```

---

## 📄 开源协议
[MIT License](LICENSE)