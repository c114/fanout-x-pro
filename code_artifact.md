#!/usr/bin/env bash

# Fanout-X Pro 一键部署脚本

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${GREEN}=== 开始安装 Fanout-X Pro 代理分发与节点提取工具 ===${NC}"

# 1. 检查 root 权限
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}错误: 请使用 root 权限运行此脚本！${NC}"
  exit 1
fi

# 2. 安装必要依赖 (Go / Curl / Git)
echo -e "${YELLOW}正在检查并安装环境依赖...${NC}"
if command -v apt-get &> /dev/null; then
    apt-get update -y && apt-get install -y curl git golang-go
elif command -v yum &> /dev/null; then
    yum install -y curl git golang
fi

# 3. 创建工作目录
WORK_DIR="/opt/fanout-x-pro"
mkdir -p ${WORK_DIR}
cd ${WORK_DIR}

# 4. 写入源码并编译
echo -e "${YELLOW}正在编译 Fanout-X Pro 二进制文件...${NC}"

# 此处在 GitHub 部署时替换为 git clone 源码
cat << 'EOF' > main.go
$(cat main.go)
EOF

go build -o fanout-x-pro main.go

# 5. 配置 Systemd 服务
echo -e "${YELLOW}配置 Systemd 服务...${NC}"
cat << EOF > /etc/systemd/system/fanout-x-pro.service
[Unit]
Description=Fanout-X Pro Proxy Controller Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${WORK_DIR}
ExecStart=${WORK_DIR}/fanout-x-pro -port 8080
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 6. 启动服务
systemctl daemon-reload
systemctl enable fanout-x-pro
systemctl restart fanout-x-pro

echo -e "${GREEN}==================================================${NC}"
echo -e "${GREEN}🎉 Fanout-X Pro 安装并启动成功！${NC}"
echo -e "${GREEN}🌐 Web 控制台访问地址: http://$(curl -s ifconfig.me):8080${NC}"
echo -e "${GREEN}📋 SOCKS5 提取链接: http://$(curl -s ifconfig.me):8080/api/v1/extract/proxies?type=socks5${NC}"
echo -e "${GREEN}==================================================${NC}"