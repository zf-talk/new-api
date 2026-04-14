# Docker 部署指南

## 前置要求

- 服务器已安装 Docker 和 Docker Compose
- 服务器已安装 Git（可选，也可以直接上传代码）

## 部署步骤

### 第 1 步：上传代码到服务器

把整个项目目录打包上传到服务器：

```bash
# 本地打包（排除不需要的文件）
# 在项目根目录 new-api-main 的上级目录执行
tar --exclude='node_modules' --exclude='.git' --exclude='data' --exclude='logs' -czf new-api.tar.gz new-api-main

# 上传到服务器
scp new-api.tar.gz root@你的服务器IP:/opt/
```

### 第 2 步：服务器上解压

SSH 登录服务器后执行：

```bash
cd /opt
tar -xzf new-api.tar.gz
cd new-api-main
```

### 第 3 步：在服务器上构建 Docker 镜像

```bash
docker build -t altronsoft/new-api:latest .
```

构建过程会自动完成前端编译（bun）和后端编译（go），大约需要 5-10 分钟。

### 第 4 步：修改 docker-compose.yml

```bash
vi docker-compose.yml
```

需要修改的内容：

1. 确认 `image: altronsoft/new-api:latest`
2. 把 PostgreSQL 密码 `123456` 改为你自己的强密码
3. 同步修改 `SQL_DSN` 中的密码

建议同时在 `.env` 中设置数据根目录（`docker-compose.yml` 已支持）：

```bash
# 所有持久化数据会保存到该目录下
DATA_DIR=/opt/new-api-data
```

### 第 5 步：启动服务

```bash
docker compose up -d
```

### 第 6 步：检查运行状态

```bash
# 查看容器状态
docker compose ps

# 查看 new-api 日志
docker compose logs -f new-api
```

### 第 7 步：访问并配置

1. 浏览器访问 `http://服务器IP:3000`
2. 创建管理员账号
3. 进入管理后台 → 支付设置 → USDT 设置
4. 填写 BEpusdt API 地址、API 认证令牌等配置
5. 确保「系统设置 → 服务器地址」填写了正确的外网地址（如 `https://your-domain.com`），回调地址依赖此配置

## 后续更新

代码修改后重新部署：

```bash
cd /opt/new-api-main
# 上传新代码后
docker build -t altronsoft/new-api:latest .
docker compose up -d
```

## 常用命令

```bash
# 重启
docker compose restart new-api

# 停止所有服务
docker compose down

# 查看实时日志
docker compose logs -f new-api

# 进入容器排查问题
docker exec -it new-api sh
```

## 启用 BEpusdt USDT 支付（可选）

[BEpusdt](https://github.com/altronsoft/bepusdt) 是一个开源 USDT 收款网关，完全兼容 Epusdt API，支持 TRON、Ethereum、BSC 等主流网络。

### 前置准备

创建数据目录（所有容器数据都挂载到本地目录）：

```bash
mkdir -p "${DATA_DIR:-.}"/{data,logs,postgres/data,bepusdt/data}
# 如果使用 MySQL，也需要创建：
# mkdir -p "${DATA_DIR:-.}"/mysql/data
```

### 步骤 1：取消注释 docker-compose.yml 中的 bepusdt 服务

```yaml
  bepusdt:
    image: altronsoft/bepusdt:latest
    container_name: bepusdt
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ${DATA_DIR:-.}/bepusdt/data:/var/lib/bepusdt
    networks:
      - new-api-network
```

同时取消注释 new-api 的 `depends_on` 下的 `- bepusdt`。

### 步骤 2：启动所有服务

```bash
docker compose up -d
```

### 步骤 3：完成 BEpusdt 首次配置

浏览器访问 `http://服务器IP:8080`，完成初始设置（添加收款地址、设置 API Token 等）。

> **生产环境**：建议通过 Nginx 反代，不要将 8080 端口直接暴露到公网；后台「应用 URI」填写可外网访问的 HTTPS 地址，BEpusdt 回调才能正常到达 new-api。

### 步骤 4：在 new-api 管理后台配置 USDT 支付

进入 new-api 管理后台 → 设置 → 支付设置 → USDT 设置，填写：

| 配置项 | 值 |
|--------|-----|
| API 地址 | `http://bepusdt:8080`（同一 Docker 网络内直连） |
| API Token | BEpusdt 后台 → 系统管理 → 基本设置 → API 设置 → 对接令牌 |
| 法币货币代码 | `CNY`（或 `USD` 等） |
| 区块链网络 | `tron`（默认，支持 TRC20） |
| 最低充值金额 | 按需设置 |

### 服务间通信说明

new-api 和 bepusdt 处于同一 Docker 网络（`new-api-network`），因此：

- new-api → BEpusdt：使用内网地址 `http://bepusdt:8080`，无需暴露端口
- BEpusdt → new-api 回调：BEpusdt 使用创建订单时传入的 `notify_url`，该地址由 new-api 的「系统设置 → 服务器地址」决定，**必须是 BEpusdt 容器可以访问到的外网地址**

## 生产环境建议

1. 使用 Nginx 反向代理并配置 HTTPS
2. BEpusdt 回调地址必须是外网可访问的 HTTPS 地址
3. 修改所有默认密码
4. 设置 `SESSION_SECRET` 环境变量为随机字符串
5. 定期备份 PostgreSQL 数据
6. BEpusdt 的 8080 端口建议仅内网访问，通过 Nginx 以域名对外提供服务
