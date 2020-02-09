package models

// LiveInfo 存储在redis中的流播放信息
type LiveInfo struct {
	Rtmp    string `json:"rtmp,omitempty"`
	Flv     string `json:"flv,omitempty"`
	Hls     string `json:"hls,omitempty"`
	Name    string `json:"name,omitempty"`
	Appname string `json:"appname,omitempty"`
	LiveId  string `json:"live_id,omitempty"`
}

// PlayLive 播放控制消息对象
type PlayLive struct {
	Appname  string `json:"appname,omitempty"`
	LiveId   string `json:"live_id,omitempty"`
	Push     string `json:"push,omitempty"`
	DeviceId string `json:"device_id,omitempty"`
}
