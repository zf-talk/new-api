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
docker build -t new-api:latest .
```

构建过程会自动完成前端编译（bun）和后端编译（go），大约需要 5-10 分钟。

### 第 4 步：修改 docker-compose.yml

```bash
vi docker-compose.yml
```

需要修改的内容：

1. 把 `image: calciumion/new-api:latest` 改为 `image: new-api:latest`
2. 把 PostgreSQL 密码 `123456` 改为你自己的强密码
3. 同步修改 `SQL_DSN` 中的密码

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
docker build -t new-api:latest .
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

## 生产环境建议

1. 使用 Nginx 反向代理并配置 HTTPS
2. BEpusdt 回调地址必须是外网可访问的 HTTPS 地址
3. 修改所有默认密码
4. 设置 `SESSION_SECRET` 环境变量为随机字符串
5. 定期备份 PostgreSQL 数据
