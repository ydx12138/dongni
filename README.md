# 懂你 Blog 后端

基于 Go、Gin 与 GORM 构建的个人博客 API 服务，为 PC 博客前端提供内容、认证、互动和后台管理能力，也保留了微信小程序登录扩展接口。

## 核心能力

- 文章、分类、评论与点赞
- 邮箱验证码注册、账号登录与密码找回
- JWT 认证与 Redis 单点登录控制
- 管理后台的文章、草稿、评论、用户、分类与网站设置管理
- OSS 图片上传与敏感词过滤
- 可配置站点名称、注册、分类、个人页、评论、社交链接和头像
- 微信登录与手机号获取的服务端封装

## 架构

项目遵循传统三层架构：

```text
handler -> service -> repository -> MySQL / Redis
```

```text
internal/
├── handler/        # HTTP 请求处理与参数绑定
├── service/        # 业务逻辑
├── repository/     # MySQL、Redis 数据访问
├── middleware/     # 鉴权与中间件
└── utils/          # 公共工具
models/
├── dto/            # 请求模型
└── vo/             # 响应模型
pkg/code/           # 业务响应码
```

## 技术栈

- Go 1.25
- Gin
- GORM
- MySQL
- Redis
- JWT
- Viper
- Zap
- Gomail
- 阿里云 OSS

## 本地运行

运行前请准备 Go 1.25、MySQL 8 与 Redis 7，并在项目根目录创建自己的 `settings.yaml` 配置文件。

```bash
go mod download
go run .
```

服务默认监听 `8080` 端口，可使用健康检查确认服务状态：

```bash
curl http://127.0.0.1:8080/health
```

需要初始化演示数据时：

```bash
go run . -seed
```

配置中应使用自己的数据库、Redis、邮件与对象存储参数。例如：

```yaml
mysql:
  host: "127.0.0.1"
  port: 3306
  username: "your_username"
  password: "your_password"
redis:
  host: "127.0.0.1"
  port: 6379
mail:
  host: "smtp.example.com"
  username: "your_email@example.com"
oss:
  endpoint: "your_oss_endpoint"
  access_key_id: "your_access_key_id"
  access_key_secret: "your_access_key_secret"
```

## Docker 部署

项目提供 MySQL、Redis、Go 服务与 Nginx 的 Docker Compose 编排：

```bash
docker compose up -d --build
curl http://127.0.0.1/health
```

部署前请按 `docker-compose.yml` 与 `settings.docker.yaml` 配置实际环境参数；前端构建产物可按 Compose 文件中的说明挂载到 Nginx。

## 安全说明

- 不要将真实的数据库密码、OSS 密钥、邮件授权码、短信密钥或微信 AppSecret 提交到仓库。
- 示例配置只应使用占位符；已泄露或曾被提交的密钥，应立即前往对应平台轮换。
- 当前仅微信 `AppSecret` 支持通过 `WECHAT_APP_SECRET` 环境变量覆盖，其他配置仍由 YAML 文件读取。

## 相关项目

前端项目位于同级目录：`../vue6122`。
