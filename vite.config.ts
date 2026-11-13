import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

// 报表前端（Vue 3 + Element Plus）入口配置。
// 说明：SDK 仍由 webpack 构建（npm run build:sdk），SDK 测试仍由 node:test 运行
//（npm run test:sdk）；vitest 只跑 src/__tests__ 下的前端测试。
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173
  },
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/__tests__/**/*.test.ts'],
    exclude: ['node_modules', 'dist', 'src/sdk.test.js']
  }
})
