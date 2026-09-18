import request from './request'

// 网关对消息体只校验 JSON 合法性并原样转发，
// 这里统一包成 {type, data, timestamp} 信封，与接口文档示例一致
function buildMessage(data) {
  return { type: 'text', data, timestamp: Date.now() }
}

// 全服广播：推给当前网关所有在线客户端
export function broadcastAll(data) {
  return request.post('/api/broadcast', buildMessage(data))
}

// 房间广播：只推给指定房间内的在线客户端；房间不存在或无人时返回 404
export function broadcastRoom(roomId, data) {
  return request.post(`/api/room/${encodeURIComponent(roomId)}/broadcast`, buildMessage(data))
}

// 单播：推给指定 clientId；对方离线或不存在时返回 404
export function sendToClient(clientId, data) {
  return request.post(`/api/client/${encodeURIComponent(clientId)}/send`, buildMessage(data))
}
