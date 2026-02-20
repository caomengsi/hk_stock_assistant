# 港股分析预测 Web 工具

仿照 A 股股票助手实现的**港股**分析预测工具：自选监控、大盘总结、个股 AI 预测，前端为网页（React），后端为 Go（Hertz 网关 + Kitex 微服务）。无需登录即可使用。

## 功能

- **首页**：自选港股列表、实时行情（价格、涨跌幅、成交量），支持添加/移除、下拉刷新，点击股票可跳转预测页。
- **大盘总结**：恒生指数等主要指数实时数据。
- **个股预测**：输入港股代码（如 `hk00700` 或 `700`），获取基于实时行情与可选 LLM 的走势分析与建议。

## 技术栈

- **前端**：React 18 + Vite + TypeScript，React Router，Axios；自选存 localStorage（key: `hk_watchlist`）。
- **后端**：Go 1.21+，CloudWeGo Hertz（HTTP 网关 :8080），CloudWeGo Kitex（RPC）；**港股个股实时行情**来自**东方财富 push2**（与券商/华盛通等一致、更实时），大盘指数仍来自新浪。
- **AI 预测**：可选。优先支持**智谱 AI**：设置 `ZHIPU_API_KEY` 即可（默认模型 `glm-4-flash`）；也可设置 `LLM_API_KEY` + `LLM_BASE_URL`、`LLM_MODEL` 使用其他 OpenAI 兼容接口。未设置时返回占位说明。

## 项目结构

```
hk_stock_assistant/
├── idl/                 # Thrift：stock.thrift, ai.thrift, api.thrift
├── backend/
│   ├── gateway/         # Hertz HTTP 网关，端口 8080
│   ├── stock_service/   # Kitex 股票服务，端口 8888，对接新浪港股
│   └── ai_service/      # Kitex AI 服务，端口 8889，预测逻辑
├── web/                 # React SPA
└── README.md
```

## 本地运行

### 1. 启动后端（三选二或全开）

**股票服务**（必选，网关依赖）：

```bash
cd backend/stock_service
go run .
# 监听 0.0.0.0:8888
```

**网关**（必选，前端请求入口）：

```bash
cd backend/gateway
go run .
# 监听 0.0.0.0:8080
```

**AI 服务**（可选，不启动则预测接口返回占位或需网关不调用 AI）：

```bash
cd backend/ai_service
# 推荐：使用智谱 AI，设置后即可启用真实预测
# export ZHIPU_API_KEY=你的智谱API密钥   # 在 https://open.bigmodel.cn 申请
# 或使用其他 OpenAI 兼容接口：
# export LLM_API_KEY=sk-xxx
# export LLM_BASE_URL=https://api.openai.com/v1
# export LLM_MODEL=gpt-4o-mini
go run .
# 监听 0.0.0.0:8889
```

### 2. 启动前端

```bash
cd web
npm install
npm run dev
```

浏览器访问 Vite 提供的地址（如 http://localhost:5173）。前端通过 Vite 代理将 `/api` 转发到 `http://localhost:8080`，因此需先启动网关（以及网关所依赖的 stock_service）。

### 3. 港股代码说明

- 统一格式：`hk` + 5 位数字，例如 `hk00700`（腾讯）、`hk09988`（阿里巴巴）。
- 前端输入支持简写：`700`、`00700` 会自动补全为 `hk00700`。

## API 说明

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/stocks/:code/realtime | 单只港股实时行情，code 如 hk00700 |
| GET | /api/market/summary | 大盘指数（如恒生指数） |
| POST | /api/prediction/:code | 个股预测，body: `{ "days": 3, "include_news": true, "model": "" }` |

## 配置与扩展

- **智谱 AI**：在 [智谱开放平台](https://open.bigmodel.cn) 申请 API Key 后，设置环境变量 `ZHIPU_API_KEY` 即可，默认使用 `glm-4-flash`；可选 `ZHIPU_MODEL` 指定模型（如 `glm-4`）。
- **其他 LLM**：也可通过 `LLM_API_KEY`、`LLM_BASE_URL`、`LLM_MODEL` 使用任意 OpenAI 兼容接口。
- **数据源**：港股**个股实时**来自东方财富 `push2.eastmoney.com`（与华盛通等券商数据一致）；**大盘指数**来自新浪 `int_hangseng`。更换数据源可修改 `backend/stock_service/biz/provider/eastmoney_hk/client.go`（个股）或 `sina_hk/client.go`（指数）。

## 依赖说明

- 后端仅使用公开 Go 模块（如 `github.com/cloudwego/hertz`、`github.com/cloudwego/kitex`），未使用任何内部/私有依赖。
- 前端依赖来自 npm 公共仓库。

## 首次推送到 GitHub（方式 A：HTTPS + Token）

仓库已建好且本地已 commit 后，在本机终端执行（将 `YOUR_GITHUB_TOKEN` 换成你在 GitHub 生成的 Personal Access Token）：

```bash
cd /Users/bytedance/Downloads/stoce_assistant-main/hk_stock_assistant
git remote set-url origin https://YOUR_GITHUB_TOKEN@github.com/caomengsi/hk_stock_assistant.git
git push -u origin main
```

推送成功后，建议把远程地址改回不含 Token 的地址，避免泄露：

```bash
git remote set-url origin https://github.com/caomengsi/hk_stock_assistant.git
```
