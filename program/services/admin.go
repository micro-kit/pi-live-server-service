package services

/* 提供给管理后台使用的rpc */

// Admin 实现grpc管理端rpc接口
type Admin struct {
	Base
}

// NewAdmin 创建管理后台rpc对象
func NewAdmin() *Admin {
	return &Admin{
		Base: NewBase(),
	}
}
