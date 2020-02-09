package program

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/micro-kit/micro-common/common"
	"github.com/micro-kit/micro-common/logger"
	"github.com/micro-kit/microkit-client/proto/piliveserverpb"
	"github.com/micro-kit/microkit/server"
	"github.com/micro-kit/pi-live-server-service/program/services"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Program 应用实体
type Program struct {
	srv    *server.Server
	logger *zap.SugaredLogger
}

func init() {
	envFile := common.GetRootDir() + ".env"
	if ext, _ := common.PathExists(envFile); ext == true {
		err := godotenv.Load(envFile)
		if err != nil {
			log.Println("读取.env错误", err)
		}
	}
}

// New 创建应用
func New() *Program {
	// 使用默认服务，如果自定义可设置对应参数
	srv, err := server.NewDefaultServer()
	if err != nil {
		log.Fatalln("创建grpc服务错误", err)
	}
	return &Program{
		srv:    srv,
		logger: logger.Logger,
	}
}

// Run 运行程序
func (p *Program) Run() {
	foreground := services.NewForeground()
	// 处理播放端播放消息
	go foreground.SubPlayLive()
	// 启动服务
	p.srv.Serve(func(grpcServer *grpc.Server) {
		piliveserverpb.RegisterPiLiveServerServer(grpcServer, foreground)
	})
	return
}

// Stop 程序结束要做的事
func (p *Program) Stop() {
	p.srv.Stop()
	return
}
