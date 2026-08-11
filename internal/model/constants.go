// Package model defines the data model and shared constants in the project.
package model

// WebSocket 自定义关闭码（4000-4999 为应用自定义范围）
// 这些状态码用于在 WebSocket 连接异常关闭时向客户端传达具体的拒绝原因。
const (
	// CloseCodeMissingParam 参数缺失：clientId 或 roomId 为空
	CloseCodeMissingParam = 4000
	// CloseCodeInvalidFormat 参数格式无效：clientId 或 roomId 包含非法字符
	CloseCodeInvalidFormat = 4001
	// CloseCodeDuplicateID 重复连接：clientId 已被占用
	CloseCodeDuplicateID = 4002
)

// 业务响应码（独立于 HTTP 状态码）
// 用于 HTTP API 响应体中的 code 字段, 便于客户端区分业务层面的成功与失败。
const (
	// BizCodeSuccess 业务处理成功
	BizCodeSuccess = 0
	// BizCodeBadRequest 请求参数错误（如 JSON 格式无效、字段缺失等）
	BizCodeBadRequest = 400
	// BizCodeNotFound 资源不存在（如目标房间不存在、目标客户端离线等）
	BizCodeNotFound = 404
	// BizCodeInternalError 服务器内部错误（如 WebSocket 协议升级失败）
	BizCodeInternalError = 500
)
