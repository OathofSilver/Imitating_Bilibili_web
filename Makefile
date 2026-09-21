.PHONY: help up down dev dev-web dev-server health test

DOMAINS := user video interaction comment search

# 本机只有独立版 docker-compose；若环境支持插件版，可用 make up COMPOSE="docker compose"
# 独立版会读取 server/deploy/.env（模板见同目录 .env.example），因此 compose 中不写口令字面量。
COMPOSE ?= docker-compose
COMPOSE_FILE := server/deploy/docker-compose.yaml

# 中间件地址：容器端口映射到宿主机的备用端口（见 deploy/docker-compose.yaml 注释）
export MYSQL_ADDR ?= 127.0.0.1:13306
export REDIS_ADDR ?= 127.0.0.1:6379
export RABBITMQ_ADDR ?= 127.0.0.1:5673
export ES_ADDR ?= 127.0.0.1:9200

# 口令与签名密钥不入配置文件（etc/*.yaml 只留地址等非敏感项），
# 本机开发默认值在此注入；生产环境必须通过真实环境变量覆盖。
# MYSQL_ROOT_PASSWORD 仅服务于 deploy/.env（容器侧），二者保持一致。
export MYSQL_USER ?= root
export MYSQL_DATABASE ?= bilibili_web
export MYSQL_PASSWORD ?= root
export MYSQL_ROOT_PASSWORD ?= $(MYSQL_PASSWORD)
export JWT_SECRET ?= dev-only-secret-change-in-production

# 依赖真实 MySQL 的 model 测试用该串连接；未提供时相关用例自动跳过。
export BW_TEST_MYSQL_DSN ?= $(MYSQL_USER):$(MYSQL_PASSWORD)@tcp($(MYSQL_ADDR))/$(MYSQL_DATABASE)?charset=utf8mb4&parseTime=true&loc=Local

help:
	@echo "up          启动 MySQL / Redis / RabbitMQ / Elasticsearch"
	@echo "down        停止并移除中间件容器"
	@echo "dev         启动前端与全部后端域服务"
	@echo "dev-web     仅启动前端开发服务器 (5173)"
	@echo "dev-server  仅启动五个后端域服务 (18001-18005)"
	@echo "health      逐个调用五个域的健康检查接口"
	@echo "test        运行后端全部测试（含需要 MySQL 的 model 用例）"

up:
	$(COMPOSE) -f $(COMPOSE_FILE) up -d

down:
	$(COMPOSE) -f $(COMPOSE_FILE) down

dev: dev-server dev-web

dev-web:
	pnpm -C web dev

dev-server:
	@for d in $(DOMAINS); do \
		echo "starting $$d-api"; \
		(cd server/app/$$d/api && go run . > /dev/null 2>&1 &) ; \
	done

health:
	@curl -s --max-time 3 http://127.0.0.1:18001/api/v1/user/health; echo
	@curl -s --max-time 3 http://127.0.0.1:18002/api/v1/video/health; echo
	@curl -s --max-time 3 http://127.0.0.1:18003/api/v1/interaction/health; echo
	@curl -s --max-time 3 http://127.0.0.1:18004/api/v1/comment/health; echo
	@curl -s --max-time 3 http://127.0.0.1:18005/api/v1/search/health; echo

test:
	@cd server && go test -p 1 ./...
