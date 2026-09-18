import request from './request'

// 管理员登录，成功后从 res.data.token 取 JWT 令牌
export function login(username, password) {
  return request.post('/api/auth/login', { username, password })
}
