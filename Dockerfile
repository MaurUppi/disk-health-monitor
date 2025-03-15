FROM golang:1.21-alpine

WORKDIR /app

# 安装基本工具
RUN apk add --no-cache git make

# 创建一个非root用户来进行编译（可选但推荐）
RUN adduser -D -g '' appuser
USER appuser

# 设置卷，这样我们可以挂载本地代码
VOLUME ["/app"]

# 设置工作目录
WORKDIR /app

# 容器启动时的命令
CMD ["sh"]