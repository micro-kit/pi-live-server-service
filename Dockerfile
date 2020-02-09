FROM alpine:latest
# 解决go 时区和https请求证书错误问题
RUN  apk update \
  && apk add ca-certificates \
  && update-ca-certificates \
  && apk add tzdata

# 需要挂载目录
VOLUME /data/logs
VOLUME /data/config

# 工作目录
WORKDIR /data/

# 导出端口
EXPOSE 9001

# 复制可执行程序
COPY ./bin/pi-live-server-service /data/

# 启动脚本
CMD ["./pi-live-server-service"]
