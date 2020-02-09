package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/micro-kit/micro-common/cache"
	"github.com/micro-kit/micro-common/logger"
	"github.com/micro-kit/micro-common/microerror"
	"github.com/micro-kit/microkit-client/proto/piliveserverpb"
	"github.com/micro-kit/pi-live-server-service/program/common"
	"github.com/micro-kit/pi-live-server-service/program/models"
)

/* 提供给客户端使用的rpc */

// Foreground 实现grpc客户端rpc接口
type Foreground struct {
	Base
}

// NewForeground 创建客户端rpc对象
func NewForeground() *Foreground {
	return &Foreground{
		Base: NewBase(),
	}
}

// GetLiveUrl 获取播放地址
func (s *Foreground) GetLiveUrl(ctx context.Context, req *piliveserverpb.GetLiveUrlRequest) (*piliveserverpb.GetLiveUrlReply, error) {
	// 验证参数是否错误
	if req.LiveId == "" {
		return nil, microerror.GetMicroError(10001)
	}
	// 默认live
	if req.Appname == "" {
		req.Appname = "live"
	}

	// 查询redis中是否存在指定key
	key := fmt.Sprintf(common.RedisKeyLiveId, req.Appname, req.LiveId)
	liveInfoStr, err := cache.GetClient().Get(key).Result()
	if err != cache.NilErr {
		logger.Logger.Errorw("读取redis中流信息错误", "err", err, "key", key)
		return nil, microerror.GetMicroError(10001, err)
	}
	liveInfo := new(models.LiveInfo)
	err = json.Unmarshal([]byte(liveInfoStr), liveInfo)
	if err != nil {
		logger.Logger.Warnw("解析redis存储live信息错误", "err", err, "val", liveInfoStr)
		return nil, microerror.GetMicroError(10000, err)
	}

	// 返回结果
	reply := &piliveserverpb.GetLiveUrlReply{
		Live: &piliveserverpb.LiveInfo{
			Rtmp: liveInfo.Rtmp,
			Flv:  liveInfo.Flv,
			Hls:  liveInfo.Hls,
			Name: liveInfo.Name,
		},
	}
	return reply, nil
}

// QueryLivesByApp 获取app下流列表
func (s *Foreground) QueryLivesByApp(ctx context.Context, req *piliveserverpb.QueryLivesByAppRequest) (*piliveserverpb.QueryLivesByAppReply, error) {
	// 默认live
	if req.Appname == "" {
		req.Appname = "live"
	}
	// 获取此应用下所有key
	key := fmt.Sprintf(common.RedisKeyLiveId, req.Appname, "*")
	keys, _ := cache.GetClient().Keys(key).Result()
	lives := make([]*piliveserverpb.LiveInfo, 0)
	for _, k := range keys {
		val := cache.GetClient().Get(k).Val()
		if val != "" {
			one := new(piliveserverpb.LiveInfo)
			err := json.Unmarshal([]byte(val), one)
			if err != nil {
				logger.Logger.Warnw("解析redis值为liveInfo错误", "err", err, "k", k)
				continue
			}
			one.Snapshot = fmt.Sprintf("%s/%s-%s.png", os.Getenv("SNAPSHOT_ADDR"), one.Appname, one.LiveId)
			lives = append(lives, one)
		}
	}

	// 返回结果
	reply := &piliveserverpb.QueryLivesByAppReply{
		Lives: lives,
	}
	return reply, nil
}

// HeartbeatLive 树莓派定时上报流信息
func (s *Foreground) HeartbeatLive(stream piliveserverpb.PiLiveServer_HeartbeatLiveServer) error {
	isSub := false
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			logger.Logger.Debugw("读取到流结束", "err", err)
			return nil
		}
		if err != nil {
			logger.Logger.Errorw("读取新流错误", "err", err)
			return nil
		}
		playKey := fmt.Sprintf("%s-%s", in.Appname, in.LiveId)
		// 为此客户端连接订阅推流状态
		if isSub == false {
			go func() {
				s.AddClientStream(playKey, stream) // 发送播放状态
			}()
		}
		isSub = true

		// 可供访问地址 - 域名或ip
		address := common.EnvAddress()
		// 存储流信息
		key := fmt.Sprintf(common.RedisKeyLiveId, in.Appname, in.LiveId)
		liveInfo := &models.LiveInfo{
			Rtmp:    fmt.Sprintf("rtmp://%s:1935/%s/%s", address, in.Appname, in.LiveId),
			Flv:     fmt.Sprintf("http://%s:7001/%s/%s.flv", address, in.Appname, in.LiveId),
			Hls:     fmt.Sprintf("http://%s:7002/%s/%s.m3u8", address, in.Appname, in.LiveId),
			Name:    in.Name,
			Appname: in.Appname,
			LiveId:  in.LiveId,
		}
		liveInfoBytes, _ := json.Marshal(liveInfo)
		err = cache.GetClient().Set(key, string(liveInfoBytes), time.Second*15).Err() // 每10秒上报一次 这里缓存15秒
		if err != nil {
			stream.Send(&piliveserverpb.HeartbeatLiveReply{
				Status: "error",
				Push:   s.GetPlayStatus(playKey),
			})
			continue
		}
		err = stream.Send(&piliveserverpb.HeartbeatLiveReply{
			Status: "ok",
			Push:   s.GetPlayStatus(playKey),
		})
		if err != nil {
			logger.Logger.Errorw("发送响应消息错误", "err", err)
		}
	}
}

// SnapshotPath 快照目录
var SnapshotPath = "./snapshot"

// 创建图片存储目录
func init() {
	err := os.MkdirAll(SnapshotPath, 0755)
	if err != nil {
		logger.Logger.Errorw("创建快照存储目录错误", "err", err)
		return
	}
	// 启动静态文件服务
	go func() {
		http.Handle("/", http.FileServer(http.Dir(SnapshotPath)))
		address := os.Getenv("HTTP_ADDR")
		if address == "" {
			address = ":10280"
		}
		err = http.ListenAndServe(address, nil)
		if err != nil {
			logger.Logger.Errorw("启动静态文件服务错误", "err", err, "address", address)
		}
	}()
}

// UploadSnapshot 上传快照图片
func (s *Foreground) UploadSnapshot(ctx context.Context, req *piliveserverpb.UploadSnapshotRequest) (*piliveserverpb.UploadSnapshotReply, error) {
	rsp := &piliveserverpb.UploadSnapshotReply{
		Status: "error",
	}
	if req.Appname == "" || req.LiveId == "" || len(req.Body) == 0 {
		logger.Logger.Debugw("参数错误", "appname", req.Appname, "liveid", req.LiveId)
		return rsp, microerror.GetMicroError(10002)
	}
	// 保存文件到目录
	filepath := fmt.Sprintf("%s/%s-%s.png", SnapshotPath, req.Appname, req.LiveId)
	err := ioutil.WriteFile(filepath, req.Body, 0644)
	if err != nil {
		logger.Logger.Errorw("写快照文件错误", "err", err, "filepath", filepath)
		return rsp, err
	}
	rsp.Status = "ok"
	return rsp, nil
}

// PlayLive 客户端发送要播放流指令
func (s *Foreground) PlayLive(stream piliveserverpb.PiLiveServer_PlayLiveServer) error {
	lastPlayLive := models.PlayLive{
		Appname:  "",
		LiveId:   "",
		Push:     "",
		DeviceId: "",
	}
	defer func() {
		// 函数退出，标记未不在播放
		if lastPlayLive.Appname != "" && lastPlayLive.DeviceId != "" {
			lastPlayLive.Push = PlayStatusStop
			s.pushChan <- lastPlayLive
		}
	}()
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			logger.Logger.Debugw("读取到流结束", "err", err)
			return nil
		}
		if err != nil {
			logger.Logger.Errorw("读取新流错误", "err", err)
			return nil
		}
		if in.DeviceId == "" || in.Appname == "" || in.LiveId == "" {
			logger.Logger.Warnw("播放端发送播放信息参数错误", "err", err, "in", in)
			continue
		}
		if in.Push != PlayStatusPlay && in.Push != PlayStatusStop {
			logger.Logger.Warnw("播放端发送播放信息，播放状态参数错误", "err", err, "in", in)
			continue
		}
		// 收到播放端心跳
		js, _ := json.Marshal(in)
		log.Println("收到播放端心跳", string(js))
		lastPlayLive.Appname = in.Appname
		lastPlayLive.LiveId = in.LiveId
		lastPlayLive.Push = in.Push
		lastPlayLive.DeviceId = in.DeviceId
		s.pushChan <- lastPlayLive
	}
	return nil
}
