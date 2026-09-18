import axios from 'axios'

// 统一请求封装：所有 /api 请求都走这里，自动携带 token、统一处理 401/429
const request = axios.create({
  baseURL: 'http://localhost:8080',
  timeout: 10000,
})

// 请求拦截器：自动附加 Authorization 请求头
request.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：
// 1) 业务层失败（code != 0）统一转成 rejected，调用方只需 catch
// 2) 401 说明令牌缺失/失效/过期：清除本地 token 并跳回登录页
// 3) 429 说明触发了网关 IP 限流（令牌桶：12 秒 1 个令牌、桶容量 5），给出友好提示
request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.status === 'error' || (res.code !== undefined && res.code !== 0)) {
      return Promise.reject(new Error(res.error || `请求失败（${res.code}）`))
    }
    return res
  },
  (error) => {
    const status = error.response?.status
    if (status === 401) {
      localStorage.removeItem('token')
      if (location.pathname !== '/login') {
        location.href = '/login'
      }
      return Promise.reject(new Error('登录已失效，请重新登录'))
    }
    if (status === 429) {
      return Promise.reject(new Error('请求过于频繁，请稍后再试（网关限流：12 秒约 1 次）'))
    }
    return Promise.reject(new Error(error.response?.data?.error || error.message || '网络错误'))
  }
)

export default request
