package services

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/micro-kit/micro-common/cache"
	"github.com/micro-kit/micro-common/logger"
	"github.com/micro-kit/microkit-client/proto/piliveserverpb"
	"github.com/micro-kit/pi-live-server-service/program/models"
)

/* 客户端和管理端公用逻辑 */

const (
	// PlayStatusPlay 播放
	PlayStatusPlay = "play"
	// PlayStatusStop 停止
	PlayStatusStop = "stop"
)

const (
	// RedisSubLiveListKey 某个流订阅列表
	RedisSubLiveListKey = "live:subs:%s"
)

// Base 基础服务对象
type Base struct {
	pushChan      chan models.PlayLive
	playStatus    sync.Map
	playCounts    map[string]map[string][]string
	playCountLock sync.Mutex
	clientStreams sync.Map
}

var (
	// 保证前后台服务使用一个base
	base *Base
)

// NewBase 创建基础服务对象
func NewBase() Base {
	if base != nil {
		return *base
	}
	base = &Base{
		pushChan:      make(chan models.PlayLive, 0),
		playStatus:    sync.Map{},
		playCounts:    make(map[string]map[string][]string, 0),
		playCountLock: sync.Mutex{},
		clientStreams: sync.Map{},
	}
	return *base
}

// GetPlayStatus 获取一个流的播放状态
func (bs *Base) GetPlayStatus(key string) string {
	if val, ok := bs.playStatus.Load(key); ok {
		return val.(string)
	}
	return PlayStatusStop
}

// AddClientStream 添加一个推流客户端连接
func (bs *Base) AddClientStream(key string, stream piliveserverpb.PiLiveServer_HeartbeatLiveServer) {
	bs.clientStreams.Store(key, stream)
}

// SendToPushStream 给指定推流端发送推流状态心跳
func (bs *Base) SendToPushStream(key string) {
	if c, ok := bs.clientStreams.Load(key); ok {
		err := c.(piliveserverpb.PiLiveServer_HeartbeatLiveServer).Send(&piliveserverpb.HeartbeatLiveReply{
			Status: "ok",
			Push:   bs.GetPlayStatus(key),
		})
		if err != nil {
			logger.Logger.Errorw("发送推流状态给推流客户端，错误", "key", key, "err", err, "status", bs.GetPlayStatus(key))
		}
	} else {
		logger.Logger.Errorw("发送推流状态给推流客户端，未找到客户端", "key", key, "status", bs.GetPlayStatus(key))
	}
}

// AddPlayLive 播放端发送播放或停止消息处理 - 这里只在内存缓存，如果需要多服务共享状态请改为redis
func (bs *Base) AddPlayLive(msg models.PlayLive) {
	bs.playCountLock.Lock()
	defer bs.playCountLock.Unlock()
	// 一个直播唯一标识记录是否存在
	key := fmt.Sprintf("%s-%s", msg.Appname, msg.LiveId)
	if _, ok := bs.playCounts[key]; ok == false {
		bs.playCounts[key] = make(map[string][]string, 0)
	}
	// 记录播放状态key是否存在
	if _, ok := bs.playCounts[key][PlayStatusPlay]; ok == false {
		bs.playCounts[key][PlayStatusPlay] = make([]string, 0)
	}
	if _, ok := bs.playCounts[key][PlayStatusStop]; ok == false {
		bs.playCounts[key][PlayStatusPlay] = make([]string, 0)
	}
	// 删除当前客户端播放状态
	for k, v := range bs.playCounts[key] {
		deviceIdKey := -1
		for kk, vv := range v {
			if vv == msg.DeviceId {
				deviceIdKey = kk
				break
			}
		}
		// 删除数组一个设备id
		if deviceIdKey >= 0 {
			bs.playCounts[key][k] = append(bs.playCounts[key][k][:deviceIdKey], bs.playCounts[key][k][deviceIdKey+1:]...)
		}
	}
	// 设置当前设备播放状态
	bs.playCounts[key][msg.Push] = append(bs.playCounts[key][msg.Push], msg.DeviceId)
	// 打印控制台
	js, _ := json.Marshal(bs.playCounts)
	log.Println("订阅列表", string(js))
	// 记录订阅列表到redis
	err := cache.Client.Set(fmt.Sprintf(RedisSubLiveListKey, key), string(js), 11*time.Second).Err()
	if err != nil {
		logger.Logger.Errorw("记录订阅列表错误", "err", err, "key", key, "val", string(js))
	}
}

// PlayLiveCount 获取播放设备数量
func (bs *Base) getPlayStatus(key string) string {
	bs.playCountLock.Lock()
	defer bs.playCountLock.Unlock()
	if len(bs.playCounts[key][PlayStatusPlay]) > 0 {
		return PlayStatusPlay
	}
	return PlayStatusStop
}

// SubPlayLive 订阅一个推流状态消息 - 并发送给推流客户端
func (bs *Base) SubPlayLive() {
	// 订阅数据
	for {
		select {
		case push := <-bs.pushChan:
			pushKey := fmt.Sprintf("%s-%s", push.Appname, push.LiveId)
			// 添加播放设备信息到本地记录
			bs.AddPlayLive(push)
			// 更新
			bs.playStatus.Store(pushKey, bs.getPlayStatus(pushKey))
			// 发送播放状态变化
			bs.SendToPushStream(pushKey)
		}
	}
}
