BINARY_NAME=pi-live-server-service # 二进制文件名
SERVICE_NAME=pi-live-server-service # 注册服务名
COMPILE_TIME = $(shell date +"%Y%M%d%H%M%S") # 编译时间
BRANCH=`git symbolic-ref --short -q HEAD` # git提交版本
VERSION="-X main.VERSION=`cat 'version'` -X main.COMPILE_DATE=`$(COMPILE_TIME)` -X main.GIT_HASH=`git rev-parse HEAD` -s"
REGISTRY= # 私有镜像地址

default:
	@echo 'Usage of make: [ build | linux_build | windows_build ｜ run | docker_build | clean ]'

build: 
	@go build -ldflags ${VERSION} -o ./bin/${BINARY_NAME} ./

linux_build: 
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags ${VERSION} -o ./bin/${BINARY_NAME} ./

windows_build: 
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${VERSION} -o ./bin/${BINARY_NAME}.exe ./

docker_build: linux_build
	@docker build -t $(REGISTRY)/service/$(BINARY_NAME):$(BRANCH) .

run: build
	@SVC_NAME=${SERVICE_NAME} ./bin/${BINARY_NAME}

clean: 
	@rm -f ./bin/${BINARY_NAME}*

.PHONY: default build linux_build windows_build run docker_build clean