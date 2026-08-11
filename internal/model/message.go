// Package model defines the data model and shared constants in the project.
package model

// Message 统一消息载体
type Message struct {
	Payload []byte // 原始消息二进制内容
}
