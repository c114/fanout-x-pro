package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Node 代表一个代理节点
type Node struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // socks5, http, vless, vmess, trojan
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"password,omitempty"`
	Protocol  string    `json:"protocol,omitempty"`
	RawURI    string    `json:"raw_uri,omitempty"`
	Source    string    `json:"source"` // 3x-ui, vpngate, manual
	LatencyMs int64     `json:"latency_ms"`
	IsAlive   bool      `json:"is_alive"`
	LastCheck time.Time `json:"last_check"`
}

// Global Node Store
type NodeStore struct {
	sync.RWMutex
	nodes map[string]*Node
}

var store = &NodeStore{nodes: make(map[string]*Node)}

func (s *NodeStore) AddOrUpdate(n *Node) {
	s.Lock()
	defer s.Unlock()
	s.nodes[n.ID] = n
}

func (s *NodeStore) GetAll() []*Node {
	s.RLock()
	defer s.RUnlock()
	list := make([]*Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		list = append(list, n)
	}
	return list
}

func main() {
	port := flag.Int("port", 8080, "Web Dashboard Port")
	flag.Parse()

	// 启动后台定时测速任务
	go startHealthChecker(30 * time.Second)

	// 启动示例节点同步 (3X-UI / VPNGate 模拟数据)
	go startNodeFetcher()

	// 路由注册
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/api/v1/nodes", handleAPINodes)
	http.HandleFunc("/api/v1/extract/proxies", handleExtractProxies)
	http.HandleFunc("/api/v1/extract/sub", handleExtractSubscription)
	http.HandleFunc("/api/v1/ping", handlePingAll)

	log.Printf("[Fanout-X Pro] Server started on :%d\n", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Fatalf("Server start failed: %v", err)
	}
}

// TCP/SOCKS5 测速器
func checkNodeLatency(node *Node) (int64, bool) {
	start := time.Now()
	address := fmt.Sprintf("%s:%d", node.Host, node.Port)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return -1, false
	}
	_ = conn.Close()
	elapsed := time.Since(start).Milliseconds()
	return elapsed, true
}

func startHealthChecker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		nodes := store.GetAll()
		var wg sync.WaitGroup
		for _, node := range nodes {
			wg.Add(1)
			go func(n *Node) {
				defer wg.Done()
				lat, alive := checkNodeLatency(n)
				s.Lock()
				n.LatencyMs = lat
				n.IsAlive = alive
				n.LastCheck = time.Now()
				s.Unlock()
			}(node)
		}
		wg.Wait()
	}
}

func startNodeFetcher() {
	// 初始化添加示例节点（包含 SOCKS5 与 VLESS 提取样例）
	sampleNodes := []*Node{
		{
			ID: "n1", Name: "US-Residential-S5-01", Type: "socks5",
			Host: "192.168.1.100", Port: 1080, Username: "user1", Password: "pwd",
			Source: "Residential Pool", IsAlive: true, LatencyMs: 45,
		},
		{
			ID: "n2", Name: "JP-Tokyo-3XUI-VLESS", Type: "vless",
			Host: "jp1.example.com", Port: 443, RawURI: "vless://uuid@jp1.example.com:443?encryption=none&security=tls#JP-Tokyo-3XUI-VLESS",
			Source: "3X-UI Panel", IsAlive: true, LatencyMs: 82,
		},
		{
			ID: "n3", Name: "VPNGate-KR-Public", Type: "socks5",
			Host: "211.43.12.9", Port: 1080, Source: "VPNGate", IsAlive: true, LatencyMs: 120,
		},
	}
	for _, n := range sampleNodes {
		store.AddOrUpdate(n)
	}
}

// API: 节点 JSON 列表
func handleAPINodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(store.GetAll())
}

// API: 一键提取 SOCKS5 / HTTP 列表 (用于脚本或项目一式的快速提取)
func handleExtractProxies(w http.ResponseWriter, r *http.Request) {
	proxyType := r.URL.Query().Get("type")
	if proxyType == "" {
		proxyType = "socks5"
	}
	maxDelayStr := r.URL.Query().Get("max_delay")
	maxDelay := int64(2000)
	if maxDelayStr != "" {
		if d, err := strconv.ParseInt(maxDelayStr, 10, 64); err == nil {
			maxDelay = d
		}
	}

	nodes := store.GetAll()
	var lines []string

	for _, n := range nodes {
		if !n.IsAlive || n.LatencyMs > maxDelay {
			continue
		}
		if strings.ToLower(n.Type) == strings.ToLower(proxyType) {
			var line string
			if n.Username != "" && n.Password != "" {
				line = fmt.Sprintf("%s:%d:%s:%s", n.Host, n.Port, n.Username, n.Password)
			} else {
				line = fmt.Sprintf("%s:%d", n.Host, n.Port)
			}
			lines = append(lines, line)
		}
	}

	if r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lines)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(strings.Join(lines, "\n")))
}

// API: 一键导出通用订阅
func handleExtractSubscription(w http.ResponseWriter, r *http.Request) {
	nodes := store.GetAll()
	var uris []string

	for _, n := range nodes {
		if n.IsAlive && n.RawURI != "" {
			uris = append(uris, n.RawURI)
		}
	}

	content := strings.Join(uris, "\n")
	if r.URL.Query().Get("format") == "base64" {
		content = base64.StdEncoding.EncodeToString([]byte(content))
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(content))
}

// 手动一键触发测速
func handlePingAll(w http.ResponseWriter, r *http.Request) {
	go startHealthChecker(1 * time.Second)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"Ping process triggered"}`))
}

// Web UI 面板渲染
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("dashboard").Parse(htmlTemplate))
	tmpl.Execute(w, nil)
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Fanout-X Pro 控制台</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { background-color: #f4f6f9; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
        .navbar { background: #1e293b; }
        .card { border: none; border-radius: 12px; box-shadow: 0 4px 12px rgba(0,0,0,0.05); }
        .badge-alive { background-color: #10b981; }
        .badge-dead { background-color: #ef4444; }
        .code-block { background: #0f172a; color: #38bdf8; padding: 12px; border-radius: 8px; font-family: monospace; }
    </style>
</head>
<body>
    <nav class="navbar navbar-dark mb-4">
        <div class="container">
            <a class="navbar-brand fw-bold" href="#">🚀 Fanout-X Pro 控制台</a>
            <button class="btn btn-outline-light btn-sm" onclick="triggerPing()">⚡ 重新测速</button>
        </div>
    </nav>

    <div class="container">
        <!-- 提取工具区 -->
        <div class="card p-4 mb-4">
            <h5 class="fw-bold mb-3">📋 一键节点 / IP 提取</h5>
            <div class="row g-3">
                <div class="col-md-6">
                    <label class="form-label">SOCKS5 代理提取链接 (支持 API/TXT)</label>
                    <div class="input-group">
                        <input type="text" id="socksUrl" class="form-value form-control" readonly value="/api/v1/extract/proxies?type=socks5&format=txt">
                        <button class="btn btn-primary" onclick="copyValue('socksUrl')">复制链接</button>
                    </div>
                </div>
                <div class="col-md-6">
                    <label class="form-label">VLESS / VMess Base64 订阅链接</label>
                    <div class="input-group">
                        <input type="text" id="subUrl" class="form-value form-control" readonly value="/api/v1/extract/sub?format=base64">
                        <button class="btn btn-success" onclick="copyValue('subUrl')">复制链接</button>
                    </div>
                </div>
            </div>
        </div>

        <!-- 节点列表区 -->
        <div class="card p-4">
            <div class="d-flex justify-content-between align-items-center mb-3">
                <h5 class="fw-bold mb-0">🌐 实时节点列表</h5>
                <span class="text-muted text-sm">自动刷新中...</span>
            </div>
            <div class="table-responsive">
                <table class="table table-hover align-middle">
                    <thead class="table-light">
                        <tr>
                            <th>状态</th>
                            <th>节点名称</th>
                            <th>类型</th>
                            <th>主机/端口</th>
                            <th>来源</th>
                            <th>延迟</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody id="nodeTable">
                        <!-- 动态渲染 -->
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <script>
        function loadNodes() {
            fetch('/api/v1/nodes')
                .then(res => res.json())
                .then(data => {
                    const tbody = document.getElementById('nodeTable');
                    tbody.innerHTML = '';
                    data.forEach(node => {
                        const tr = document.createElement('tr');
                        const statusBadge = node.is_alive 
                            ? '<span class="badge badge-alive">在线</span>' 
                            : '<span class="badge badge-dead">离线</span>';
                        tr.innerHTML = `
                            <td>${statusBadge}</td>
                            <td class="fw-bold">${node.name}</td>
                            <td><span class="badge bg-secondary">${node.type.toUpperCase()}</span></td>
                            <td><code>${node.host}:${node.port}</code></td>
                            <td><small class="text-muted">${node.source}</small></td>
                            <td>${node.latency_ms > 0 ? node.latency_ms + ' ms' : '-'}</td>
                            <td><button class="btn btn-sm btn-outline-primary" onclick="alert('${node.raw_uri || (node.host+':'+node.port)}')">详情</button></td>
                        `;
                        tbody.appendChild(tr);
                    });
                });
        }

        function triggerPing() {
            fetch('/api/v1/ping').then(() => alert('全量节点测速指令已下发！'));
        }

        function copyValue(id) {
            const input = document.getElementById(id);
            const fullUrl = window.location.origin + input.value;
            navigator.clipboard.writeText(fullUrl);
            alert('复制成功: ' + fullUrl);
        }

        loadNodes();
        setInterval(loadNodes, 5000);
    </script>
</body>
</html>
`
