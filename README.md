<div align="center">

![ai-bridge](/web/public/logo.png)

# AI Bridge

🍥 **Next-Generation LLM Gateway and AI Asset Management System**

<p align="center">
  <a href="./README.zh_CN.md">简体中文</a> |
  <a href="./README.zh_TW.md">繁體中文</a> |
  <strong>English</strong> |
  <a href="./README.fr.md">Français</a> |
  <a href="./README.ja.md">日本語</a>
</p>

<p align="center">
  <a href="https://raw.githubusercontent.com/Calcium-Ion/ai-bridge/main/LICENSE">
    <img src="https://img.shields.io/github/license/Calcium-Ion/ai-bridge?color=brightgreen" alt="license">
  </a><!--
  --><a href="https://github.com/Calcium-Ion/ai-bridge/releases/latest">
    <img src="https://img.shields.io/github/v/release/Calcium-Ion/ai-bridge?color=brightgreen&include_prereleases" alt="release">
  </a><!--
  --><a href="https://hub.docker.com/r/CalciumIon/ai-bridge">
    <img src="https://img.shields.io/badge/docker-dockerHub-blue" alt="docker">
  </a><!--
  --><a href="https://goreportcard.com/report/github.com/Calcium-Ion/ai-bridge">
    <img src="https://goreportcard.com/badge/github.com/Calcium-Ion/ai-bridge" alt="GoReportCard">
  </a>
</p>

<p align="center">
  <a href="https://trendshift.io/repositories/20180" target="_blank">
    <img src="https://trendshift.io/api/badge/repositories/20180" alt="QuantumNous%2Fai-bridge | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/>
  </a>
  <br>
  <a href="https://hellogithub.com/repository/QuantumNous/ai-bridge" target="_blank">
    <img src="https://api.hellogithub.com/v1/widgets/recommend.svg?rid=539ac4217e69431684ad4a0bab768811&claim_uid=tbFPfKIDHpc4TzR" alt="Featured｜HelloGitHub" style="width: 250px; height: 54px;" width="250" height="54" />
  </a><!--
  --><a href="https://www.producthunt.com/products/ai-bridge/launches/ai-bridge?embed=true&utm_source=badge-featured&utm_medium=badge&utm_campaign=badge-ai-bridge" target="_blank" rel="noopener noreferrer">
    <img src="https://api.producthunt.com/widgets/embed-image/v1/featured.svg?post_id=1047693&theme=light&t=1769577875005" alt="AI Bridge - All-in-one AI asset management gateway. | Product Hunt" style="width: 250px; height: 54px;" width="250" height="54" />
  </a>
</p>

<p align="center">
  <a href="#-quick-start">Quick Start</a> •
  <a href="#-key-features">Key Features</a> •
  <a href="#-deployment">Deployment</a> •
  <a href="#-documentation">Documentation</a> •
  <a href="#-help-support">Help</a>
</p>

</div>

## 📝 Project Description

> [!IMPORTANT]
> - This project is for personal learning purposes only, with no guarantee of stability or technical support
> - Users must comply with OpenAI's [Terms of Use](https://openai.com/policies/terms-of-use) and **applicable laws and regulations**, and must not use it for illegal purposes
> - According to the [《Interim Measures for the Management of Generative Artificial Intelligence Services》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm), please do not provide any unregistered generative AI services to the public in China.

---

## 🤝 Trusted Partners

<p align="center">
  <em>No particular order</em>
</p>

<p align="center">
  <a href="https://www.cherry-ai.com/" target="_blank">
    <img src="./docs/images/cherry-studio.png" alt="Cherry Studio" height="80" />
  </a><!--
  --><a href="https://github.com/iOfficeAI/AionUi/" target="_blank">
    <img src="./docs/images/aionui.png" alt="Aion UI" height="80" />
  </a><!--
  --><a href="https://bda.pku.edu.cn/" target="_blank">
    <img src="./docs/images/pku.png" alt="Peking University" height="80" />
  </a><!--
  --><a href="https://www.compshare.cn/?ytag=GPU_yy_gh_aibridge" target="_blank">
    <img src="./docs/images/ucloud.png" alt="UCloud" height="80" />
  </a><!--
  --><a href="https://www.aliyun.com/" target="_blank">
    <img src="./docs/images/aliyun.png" alt="Alibaba Cloud" height="80" />
  </a><!--
  --><a href="https://io.net/" target="_blank">
    <img src="./docs/images/io-net.png" alt="IO.NET" height="80" />
  </a>
</p>

---

## 🙏 Special Thanks

<p align="center">
  <a href="https://www.jetbrains.com/?from=ai-bridge" target="_blank">
    <img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.png" alt="JetBrains Logo" width="120" />
  </a>
</p>

<p align="center">
  <strong>Thanks to <a href="https://www.jetbrains.com/?from=ai-bridge">JetBrains</a> for providing free open-source development license for this project</strong>
</p>

---

## 🚀 Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the project
git clone https://github.com/QuantumNous/ai-bridge.git
cd ai-bridge

# Create environment file and edit passwords
cp deploy/.env.example .env
nano .env   # ⚠️ Change DB_PASSWORD, REDIS_PASSWORD!

# Start with PostgreSQL (default)
docker compose --profile postgres up -d

# Access at http://localhost:3000 | Metrics at http://localhost:9090/metrics
```

<details>
<summary><strong>Using Docker Commands</strong></summary>

```bash
# Pull the latest image
docker pull calciumion/ai-bridge:latest

# Using SQLite (default)
docker run --name ai-bridge -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/ai-bridge:latest

# Using MySQL
docker run --name ai-bridge -d --restart always \
  -p 3000:3000 \
  -e SQL_DSN="root:123456@tcp(localhost:3306)/oneapi" \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/ai-bridge:latest
```

> **💡 Tip:** `-v ./data:/data` will save data in the `data` folder of the current directory, you can also change it to an absolute path like `-v /your/custom/path:/data`

</details>

---

🎉 After deployment is complete, visit `http://localhost:3000` to start using!

📖 For complete deployment guide with **Docker Compose** and **Kubernetes**, see [deploy/README.md](./deploy/README.md)

---

## 📚 Documentation

<div align="center">

### 📖 [Official Documentation](https://docs.aibridge.pro/en/docs) | [![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/QuantumNous/ai-bridge)

</div>

**Quick Navigation:**

| Category | Link |
|------|------|
| 🚀 Deployment Guide | [Installation Documentation](https://docs.aibridge.pro/en/docs/installation) |
| ⚙️ Environment Configuration | [Environment Variables](https://docs.aibridge.pro/en/docs/installation/config-maintenance/environment-variables) |
| 📡 API Documentation | [API Documentation](https://docs.aibridge.pro/en/docs/api) |
| ❓ FAQ | [FAQ](https://docs.aibridge.pro/en/docs/support/faq) |
| 💬 Community Interaction | [Communication Channels](https://docs.aibridge.pro/en/docs/support/community-interaction) |

---

## ✨ Key Features

> For detailed features, please refer to [Features Introduction](https://docs.aibridge.pro/en/docs/guide/wiki/basic-concepts/features-introduction)

### 🎨 Core Functions

| Feature | Description |
|------|------|
| 🎨 New UI | Modern user interface design |
| 🌍 Multi-language | Supports Simplified Chinese, Traditional Chinese, English, French, Japanese |
| 🔄 Data Compatibility | Fully compatible with the original One API database |
| 📈 Data Dashboard | Visual console and statistical analysis |
| 🔒 Permission Management | Token grouping, model restrictions, user management |

### 💰 Payment and Billing

- ✅ Online recharge (EPay, Stripe)
- ✅ Pay-per-use model pricing
- ✅ Cache billing support (OpenAI, Azure, DeepSeek, Claude, Qwen and all supported models)
- ✅ Flexible billing policy configuration

### 🔐 Authorization and Security

- 😈 Discord authorization login
- 🤖 LinuxDO authorization login
- 📱 Telegram authorization login
- 🔑 OIDC unified authentication
- 🔍 Key quota query usage (with [neko-api-key-tool](https://github.com/Calcium-Ion/neko-api-key-tool))

### 🚀 Advanced Features

**API Format Support:**
- ⚡ [OpenAI Responses](https://docs.aibridge.pro/en/docs/api/ai-model/chat/openai/create-response)
- ⚡ [OpenAI Realtime API](https://docs.aibridge.pro/en/docs/api/ai-model/realtime/create-realtime-session) (including Azure)
- ⚡ [Claude Messages](https://docs.aibridge.pro/en/docs/api/ai-model/chat/create-message)
- ⚡ [Google Gemini](https://doc.aibridge.pro/en/api/google-gemini-chat)
- 🔄 [Rerank Models](https://docs.aibridge.pro/en/docs/api/ai-model/rerank/create-rerank) (Cohere, Jina)

**Intelligent Routing:**
- ⚖️ Channel weighted random
- 🔄 Automatic retry on failure
- 🚦 User-level model rate limiting

**Observability & Reliability (NEW):**
- 🔗 **Distributed Tracing** — OpenTelemetry integration with W3C TraceContext propagation, automatic span instrumentation for HTTP and relay requests
- ⚡ **Circuit Breaker** — Per-channel error-rate-based circuit breaking with half-open probe and exponential backoff retry
- 🚨 **Alerting Engine** — Rule-based alerting with configurable thresholds, Webhook/Log/Database notification channels, cooldown protection
- 🎨 **Canary Release** — Weight-based and label-based traffic routing for safe gray deployments

**Format Conversion:**
- 🔄 **OpenAI Compatible ⇄ Claude Messages**
- 🔄 **OpenAI Compatible → Google Gemini**
- 🔄 **Google Gemini → OpenAI Compatible** - Text only, function calling not supported yet
- 🚧 **OpenAI Compatible ⇄ OpenAI Responses** - In development
- 🔄 **Thinking-to-content functionality**

**Reasoning Effort Support:**

<details>
<summary>View detailed configuration</summary>

**OpenAI series models:**
- `o3-mini-high` - High reasoning effort
- `o3-mini-medium` - Medium reasoning effort
- `o3-mini-low` - Low reasoning effort
- `gpt-5-high` - High reasoning effort
- `gpt-5-medium` - Medium reasoning effort
- `gpt-5-low` - Low reasoning effort

**Claude thinking models:**
- `claude-3-7-sonnet-20250219-thinking` - Enable thinking mode

**Google Gemini series models:**
- `gemini-2.5-flash-thinking` - Enable thinking mode
- `gemini-2.5-flash-nothinking` - Disable thinking mode
- `gemini-2.5-pro-thinking` - Enable thinking mode
- `gemini-2.5-pro-thinking-128` - Enable thinking mode with thinking budget of 128 tokens
- You can also append `-low`, `-medium`, or `-high` to any Gemini model name to request the corresponding reasoning effort (no extra thinking-budget suffix needed).

</details>

---

## 🤖 Model Support

> For details, please refer to [API Documentation - Relay Interface](https://docs.aibridge.pro/en/docs/api)

| Model Type | Description | Documentation |
|---------|------|------|
| 🤖 OpenAI-Compatible | OpenAI compatible models | [Documentation](https://docs.aibridge.pro/en/docs/api/ai-model/chat/openai/createchatcompletion) |
| 🤖 OpenAI Responses | OpenAI Responses format | [Documentation](https://docs.aibridge.pro/en/docs/api/ai-model/chat/openai/createresponse) |
| 🎨 Midjourney-Proxy | [Midjourney-Proxy(Plus)](https://github.com/novicezk/midjourney-proxy) | [Documentation](https://doc.aibridge.pro/api/midjourney-proxy-image) |
| 🎵 Suno-API | [Suno API](https://github.com/Suno-API/Suno-API) | [Documentation](https://doc.aibridge.pro/api/suno-music) |
| 🔄 Rerank | Cohere, Jina | [Documentation](https://docs.aibridge.pro/en/docs/api/ai-model/rerank/creatererank) |
| 💬 Claude | Messages format | [Documentation](https://docs.aibridge.pro/en/docs/api/ai-model/chat/createmessage) |
| 🌐 Gemini | Google Gemini format | [Documentation](https://docs.aibridge.pro/en/docs/api/ai-model/chat/gemini/geminirelayv1beta) |
| 🔧 Dify | ChatFlow mode | - |
| 🎯 Custom | Supports complete call address | - |

### 📡 Supported Interfaces

<details>
<summary>View complete interface list</summary>

- [Chat Interface (Chat Completions)](https://docs.aibridge.pro/en/docs/api/ai-model/chat/openai/createchatcompletion)
- [Response Interface (Responses)](https://docs.aibridge.pro/en/docs/api/ai-model/chat/openai/createresponse)
- [Image Interface (Image)](https://docs.aibridge.pro/en/docs/api/ai-model/images/openai/post-v1-images-generations)
- [Audio Interface (Audio)](https://docs.aibridge.pro/en/docs/api/ai-model/audio/openai/create-transcription)
- [Video Interface (Video)](https://docs.aibridge.pro/en/docs/api/ai-model/audio/openai/createspeech)
- [Embedding Interface (Embeddings)](https://docs.aibridge.pro/en/docs/api/ai-model/embeddings/createembedding)
- [Rerank Interface (Rerank)](https://docs.aibridge.pro/en/docs/api/ai-model/rerank/creatererank)
- [Realtime Conversation (Realtime)](https://docs.aibridge.pro/en/docs/api/ai-model/realtime/createrealtimesession)
- [Claude Chat](https://docs.aibridge.pro/en/docs/api/ai-model/chat/createmessage)
- [Google Gemini Chat](https://docs.aibridge.pro/en/docs/api/ai-model/chat/gemini/geminirelayv1beta)

</details>

---

## 🚢 Deployment

> [!TIP]
> **Latest Docker image:** `calciumion/ai-bridge:latest`
>
> **Full deployment guide:** [deploy/README.md](./deploy/README.md) (Docker Compose + Kubernetes)

### 📋 Deployment Methods Overview

| Method | Use Case | Complexity |
|--------|----------|------------|
| **[Docker Compose](./deploy/README.md#method-1-docker-compose)** | Development / Single server / PoC | ![Beginner](https://img.shields.io/badge/difficulty-beginner-green) |
| **[Kubernetes Helm Chart](./deploy/README.md#method-2-kubernetes-helm-chart)** | Production cluster (with HPA, PDB, Ingress) | ![Intermediate](https://img.shields.io/badge/difficulty-intermediate-yellow) |
| **[Kubernetes Standalone](./deploy/README.md#method-3-kubernetes-standalone-manifests)** | Production without Helm dependency | ![Intermediate](https://img.shields.io/badge/difficulty-intermediate-yellow) |
| **Docker Run** | Quick test / Minimal setup | ![Beginner](https://img.shields.io/badge/difficulty-beginner-green) |

### 📋 Deployment Requirements

| Component | Requirement |
|------|------|
| **Local database** | SQLite (Docker must mount `/data` directory)|
| **Remote database** | MySQL ≥ 5.7.8 or PostgreSQL ≥ 9.6 |
| **Container engine** | Docker / Docker Compose |
| **Orchestrator (K8s)** | Kubernetes v1.24+ / Helm 3.12+ |

### ⚙️ Environment Variable Configuration

<details>
<summary>Common environment variable configuration</summary>

| Variable Name | Description | Default Value |
|--------|------|--------|
| `SESSION_SECRET` | Session secret (required for multi-machine deployment) | - |
| `CRYPTO_SECRET` | Encryption secret (required for Redis) | - |
| `SQL_DSN` | Database connection string | - |
| `REDIS_CONN_STRING` | Redis connection string | - |
| `STREAMING_TIMEOUT` | Streaming timeout (seconds) | `300` |
| `STREAM_SCANNER_MAX_BUFFER_MB` | Max per-line buffer (MB) for the stream scanner; increase when upstream sends huge image/base64 payloads | `64` |
| `MAX_REQUEST_BODY_MB` | Max request body size (MB, counted **after decompression**; prevents huge requests/zip bombs from exhausting memory). Exceeding it returns `413` | `32` |
| `AZURE_DEFAULT_API_VERSION` | Azure API version | `2025-04-01-preview` |
| `ERROR_LOG_ENABLED` | Error log switch | `false` |
| `PYROSCOPE_URL` | Pyroscope server address | - |
| `PYROSCOPE_APP_NAME` | Pyroscope application name | `ai-bridge` |
| `PYROSCOPE_BASIC_AUTH_USER` | Pyroscope basic auth user | - |
| `PYROSCOPE_BASIC_AUTH_PASSWORD` | Pyroscope basic auth password | - |
| `PYROSCOPE_MUTEX_RATE` | Pyroscope mutex sampling rate | `5` |
| `PYROSCOPE_BLOCK_RATE` | Pyroscope block sampling rate | `5` |
| `HOSTNAME` | Hostname tag for Pyroscope | `ai-bridge` |

**OpenTelemetry Tracing (NEW):**
| `OTEL_ENABLED` | Enable OpenTelemetry distributed tracing | `false` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP exporter endpoint | `localhost:4318` |
| `OTEL_SERVICE_NAME` | Service name for tracing | `ai-bridge` |
| `OTEL_SAMPLING_RATIO` | Trace sampling ratio (0.0-1.0) | `1.0` |

**Circuit Breaker (NEW):**
| `CIRCUIT_BREAKER_ENABLED` | Enable circuit breaker protection | `true` |
| `CIRCUIT_BREAKER_ERROR_THRESHOLD` | Error rate threshold to trip breaker (0.0-1.0) | `0.50` |
| `CIRCUIT_BREAKER_CONSECUTIVE_FAILURES` | Consecutive failures to trip | `5` |
| `CIRCUIT_BREAKER_TIMEOUT_SECONDS` | Open→HalfOpen wait time (seconds) | `30` |
| `CIRCUIT_BREAKER_HALF_OPEN_MAX_REQUESTS` | Max probe requests in half-open state | `3` |

**Alerting System (NEW):**
| `ALERTING_ENABLED` | Enable alerting engine | `true` |
| `ALERTING_WEBHOOK_URL` | Webhook URL for alert notifications | `` |
| `ALERTING_COOLDOWN_SECONDS` | Alert cooldown in seconds | `300` |
| `ALERTING_QUOTA_THRESHOLD` | Low quota alert threshold | `10000` |

**Canary Release (NEW):**
| `CANARY_ENABLED` | Enable canary/gray release routing | `false` |

**Prometheus Metrics (built-in):**
| `/metrics` | Prometheus scraping endpoint (auto-registered) | - |

📖 **Complete configuration:** [Environment Variables Documentation](https://docs.aibridge.pro/en/docs/installation/config-maintenance/environment-variables)

</details>

### 🔧 Deployment Methods

<details>
<summary><strong>Method 1: Docker Compose (Recommended)</strong></summary>

```bash
# Clone the project
git clone https://github.com/QuantumNous/ai-bridge.git
cd ai-bridge

# Create environment file (EDIT passwords before deploying!)
cp deploy/.env.example .env
vim .env

# Start with PostgreSQL (default)
docker compose --profile postgres up -d

# Or start with MySQL instead
docker compose --profile mysql up -d
```

**Features included:** PostgreSQL/MySQL + Redis, Prometheus metrics (`:9090/metrics`), resource limits, health checks, structured logging.

> Full guide: [deploy/README.md](./deploy/README.md#method-1-docker-compose)

</details>

<details>
<summary><strong>Method 2: Kubernetes</strong></summary>

**Helm Chart (recommended for production):**
```bash
helm install ai-bridge ./deploy/k8s/helm -n ai-bridge \
  --set sessionSecret=$(openssl rand -hex 32) \
  --set database.password="your-db-password" \
  --set redis.auth.password="your-redis-password"
```

**Standalone manifests (without Helm):**
```bash
# Edit secrets in deploy/k8s/standalone/k8s-deployment.yaml first!
kubectl apply -f deploy/k8s/standalone/k8s-deployment.yaml
kubectl port-forward svc/ai-bridge -n ai-bridge 3000:3000
```

**K8s features:** HPA auto-scaling (2-20 pods), PodDisruptionBudget, NetworkPolicy, ServiceMonitor, init containers for DB/Redis readiness.

> Full guide: [deploy/README.md](./deploy/README.md#method-2-kubernetes-helm-chart)

</details>

<details>
<summary><strong>Method 3: Docker Commands (Quick Test)</strong></summary>

**Using SQLite:**
```bash
docker run --name ai-bridge -d --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/ai-bridge:latest
```

**Using MySQL:**
```bash
docker run --name ai-bridge -d --restart always \
  -p 3000:3000 \
  -e SQL_DSN="root:123456@tcp(localhost:3306)/oneapi" \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  calciumion/ai-bridge:latest
```

> **💡 Path explanation:**
> - `./data:/data` - Relative path, data saved in the data folder of the current directory
> - You can also use absolute path, e.g.: `/your/custom/path:/data`

</details>

<details>
<summary><strong>Method 3: BaoTa Panel</strong></summary>

1. Install BaoTa Panel (≥ 9.2.0 version)
2. Search for **AI-Bridge** in the application store
3. One-click installation

📖 [Tutorial with images](./docs/BT.md)

</details>

### ⚠️ Multi-machine Deployment Considerations

> [!WARNING]
> - **Must set** `SESSION_SECRET` - Otherwise login status inconsistent
> - **Shared Redis must set** `CRYPTO_SECRET` - Otherwise data cannot be decrypted

### 🔄 Channel Retry and Cache

**Retry configuration:** `Settings → Operation Settings → General Settings → Failure Retry Count`

**Cache configuration:**
- `REDIS_CONN_STRING`: Redis cache (recommended)
- `MEMORY_CACHE_ENABLED`: Memory cache

---

## 🔗 Related Projects

### Upstream Projects

| Project | Description |
|------|------|
| [One API](https://github.com/songquanpeng/one-api) | Original project base |
| [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy) | Midjourney interface support |

### Supporting Tools

| Project | Description |
|------|------|
| [neko-api-key-tool](https://github.com/Calcium-Ion/neko-api-key-tool) | Key quota query tool |
| [ai-bridge-horizon](https://github.com/Calcium-Ion/ai-bridge-horizon) | AI Bridge high-performance optimized version |

---

## 💬 Help Support

### 📖 Documentation Resources

| Resource | Link |
|------|------|
| 📘 FAQ | [FAQ](https://docs.aibridge.pro/en/docs/support/faq) |
| 💬 Community Interaction | [Communication Channels](https://docs.aibridge.pro/en/docs/support/community-interaction) |
| 🐛 Issue Feedback | [Issue Feedback](https://docs.aibridge.pro/en/docs/support/feedback-issues) |
| 📚 Complete Documentation | [Official Documentation](https://docs.aibridge.pro/en/docs) |

### 🤝 Contribution Guide

Welcome all forms of contribution!

- 🐛 Report Bugs
- 💡 Propose New Features
- 📝 Improve Documentation
- 🔧 Submit Code

---

## 📜 License

This project is licensed under the [GNU Affero General Public License v3.0 (AGPLv3)](./LICENSE).

This is an open-source project developed based on [One API](https://github.com/songquanpeng/one-api) (MIT License).

If your organization's policies do not permit the use of AGPLv3-licensed software, or if you wish to avoid the open-source obligations of AGPLv3, please contact us at: [support@quantumnous.com](mailto:support@quantumnous.com)

---

## 🌟 Star History

<div align="center">

[![Star History Chart](https://api.star-history.com/svg?repos=Calcium-Ion/ai-bridge&type=Date)](https://star-history.com/#Calcium-Ion/ai-bridge&Date)

</div>

---

<div align="center">

### 💖 Thank you for using AI Bridge

If this project is helpful to you, welcome to give us a ⭐️ Star！

**[Official Documentation](https://docs.aibridge.pro/en/docs)** • **[Issue Feedback](https://github.com/Calcium-Ion/ai-bridge/issues)** • **[Latest Release](https://github.com/Calcium-Ion/ai-bridge/releases)**

<sub>Built with ❤️ by QuantumNous</sub>

</div>
