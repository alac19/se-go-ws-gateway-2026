<script setup>
import AppHeader from '../components/AppHeader.vue'
</script>

<template>
  <div class="page">
    <AppHeader title="系统架构图" />

    <main class="content">
      <section class="panel">
        <svg viewBox="0 0 960 660" class="arch" role="img" aria-label="WebSocket 网关系统架构图">
          <!-- ===== 管理端 ===== -->
          <rect x="310" y="16" width="340" height="56" rx="10" fill="#409eff" />
          <text x="480" y="40" text-anchor="middle" fill="#fff" font-size="15" font-weight="600">
            管理端（浏览器）· 本看板
          </text>
          <text x="480" y="60" text-anchor="middle" fill="#ecf5ff" font-size="11">
            Vue3 + Vite · 登录 / 看板 / 广播操作
          </text>

          <!-- 管理端 -> 网关 箭头 -->
          <line x1="480" y1="72" x2="480" y2="122" stroke="#606266" stroke-width="2" />
          <polygon points="474,116 486,116 480,126" fill="#606266" />
          <text x="492" y="94" fill="#606266" font-size="11">HTTP REST（Authorization: Bearer JWT）</text>
          <text x="492" y="110" fill="#606266" font-size="11">WebSocket（token 走 query 参数）</text>

          <!-- ===== 网关容器 ===== -->
          <rect x="56" y="130" width="848" height="330" rx="12" fill="#f0f6ff" stroke="#409eff" stroke-width="2" />
          <text x="480" y="154" text-anchor="middle" fill="#1f2d3d" font-size="15" font-weight="700">
            WebSocket 实时推送网关（Go · Gin · :8080）
          </text>

          <!-- 中间件层 -->
          <text x="76" y="180" fill="#909399" font-size="11">中间件 internal/middleware</text>
          <rect x="76" y="188" width="256" height="40" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="204" y="213" text-anchor="middle" fill="#303133" font-size="12">JWT 鉴权（pkg/auth · HS256 · 24h）</text>
          <rect x="352" y="188" width="256" height="40" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="480" y="213" text-anchor="middle" fill="#303133" font-size="12">IP 限流（pkg/limiter · 令牌桶 12s/1 · 容量 5）</text>
          <rect x="628" y="188" width="256" height="40" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="756" y="213" text-anchor="middle" fill="#303133" font-size="12">CORS 跨域支持</text>

          <!-- 处理器层 -->
          <text x="76" y="252" fill="#909399" font-size="11">处理器 internal/handler</text>
          <rect x="76" y="260" width="192" height="40" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="172" y="285" text-anchor="middle" fill="#303133" font-size="12">POST /api/auth/login</text>
          <rect x="284" y="260" width="192" height="40" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="380" y="285" text-anchor="middle" fill="#303133" font-size="12">GET /ws 接入（升级 101）</text>
          <rect x="492" y="260" width="192" height="40" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="588" y="285" text-anchor="middle" fill="#303133" font-size="12">广播 / 房间 / 单播 API</text>
          <rect x="700" y="260" width="184" height="40" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="792" y="285" text-anchor="middle" fill="#303133" font-size="12">GET /api/stats 统计</text>

          <!-- 服务层 -->
          <text x="76" y="324" fill="#909399" font-size="11">服务层 internal/service</text>
          <rect x="76" y="332" width="256" height="56" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="204" y="354" text-anchor="middle" fill="#303133" font-size="12">连接管理</text>
          <text x="204" y="372" text-anchor="middle" fill="#909399" font-size="10.5">注册/注销 · 心跳 Ping/Pong · clientId 身份绑定</text>
          <rect x="352" y="332" width="256" height="56" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="480" y="354" text-anchor="middle" fill="#303133" font-size="12">房间管理</text>
          <text x="480" y="372" text-anchor="middle" fill="#909399" font-size="10.5">加入/离开 · 空房自动回收</text>
          <rect x="628" y="332" width="256" height="56" rx="6" fill="#fff" stroke="#b3d8ff" />
          <text x="756" y="354" text-anchor="middle" fill="#303133" font-size="12">消息推送</text>
          <text x="756" y="372" text-anchor="middle" fill="#909399" font-size="10.5">全服 / 房间 / 单播 · 原样转发</text>

          <!-- 基础库 -->
          <text x="76" y="412" fill="#909399" font-size="11">基础库 pkg/ · 配置 internal/config</text>
          <rect x="240" y="418" width="480" height="30" rx="6" fill="#fbfdff" stroke="#dcdfe6" />
          <text x="480" y="437" text-anchor="middle" fill="#606266" font-size="11">
            pkg/auth · pkg/limiter · pkg/metrics(Prometheus) · pkg/logger ｜ configs/config.toml + WS_* 环境变量
          </text>

          <!-- 网关 -> 客户端 箭头 -->
          <line x1="480" y1="460" x2="480" y2="510" stroke="#606266" stroke-width="2" />
          <polygon points="474,504 486,504 480,514" fill="#606266" />
          <text x="492" y="482" fill="#606266" font-size="11">WebSocket 长连接（服务端每 30s Ping，60s 内须回 Pong）</text>
          <text x="492" y="498" fill="#606266" font-size="11">下行消息为 JSON 文本帧 · 上行仅维持活跃</text>

          <!-- ===== 客户端 ===== -->
          <rect x="56" y="518" width="848" height="90" rx="12" fill="#fff" stroke="#dcdfe6" stroke-width="2" />
          <text x="480" y="544" text-anchor="middle" fill="#1f2d3d" font-size="14" font-weight="600">
            WebSocket 客户端（浏览器标签页 / 演示脚本等）
          </text>
          <rect x="106" y="558" width="170" height="32" rx="6" fill="#ecf5ff" />
          <text x="191" y="578" text-anchor="middle" fill="#409eff" font-size="11">room1 · zhangsan-bot1</text>
          <rect x="296" y="558" width="170" height="32" rx="6" fill="#ecf5ff" />
          <text x="381" y="578" text-anchor="middle" fill="#409eff" font-size="11">room1 · zhangsan-bot2</text>
          <rect x="486" y="558" width="170" height="32" rx="6" fill="#ecf5ff" />
          <text x="571" y="578" text-anchor="middle" fill="#409eff" font-size="11">room2 · zhangsan-bot3</text>
          <rect x="676" y="558" width="208" height="32" rx="6" fill="#ecf5ff" />
          <text x="780" y="578" text-anchor="middle" fill="#409eff" font-size="11">…（clientId = 账号名-后缀）</text>
        </svg>
      </section>

      <section class="notes">
        <h3>说明</h3>
        <ul>
          <li>登录接口 <code>POST /api/auth/login</code> 换取 JWT 令牌；之后所有 <code>/api</code> 请求在请求头携带 <code>Authorization: Bearer &lt;token&gt;</code>，WebSocket 通过 query 参数 <code>token</code> 传递（浏览器 WS 不支持自定义请求头）。</li>
          <li>令牌桶限流仅作用于 <code>/api</code> 路由组；<code>/health</code> 健康检查与 <code>/metrics</code> 监控指标不在组内，不参与限流。</li>
          <li>网关不解析、不修改消息内容：下行推送即调用方提交的原始 JSON。</li>
          <li>JWT 令牌有效期 24 小时，过期后所有受保护接口返回 401，前端统一清除本地令牌并跳回登录页。</li>
        </ul>
      </section>
    </main>
  </div>
</template>

<style scoped>
.page {
  min-height: 100vh;
}

.content {
  padding: 24px;
}

.panel {
  padding: 20px;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.06);
}

.arch {
  display: block;
  width: 100%;
  height: auto;
}

.notes {
  margin-top: 16px;
  padding: 20px 24px;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.06);
}

.notes h3 {
  margin-bottom: 10px;
  font-size: 15px;
  color: #303133;
}

.notes li {
  margin: 6px 0 6px 18px;
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
}

code {
  padding: 1px 6px;
  background: #f0f2f5;
  border-radius: 4px;
  font-size: 12px;
  color: #476582;
}
</style>
