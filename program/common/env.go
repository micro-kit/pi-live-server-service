package common

import "os"

/* 公共获取环境变量 */

// EnvAddress 获取服务器流播放地址
func EnvAddress() string {
	return os.Getenv("Address")
}
