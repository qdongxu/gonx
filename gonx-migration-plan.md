# Gonx 迁移计划草案

> **项目**: Gonx — 将 Nginx 从 C 迁移至 Go  
> **目标仓库**: https://github.com/qdongxu/gonx  
> **分析日期**: 2026-05-30  
> **阶段**: 分析阶段（禁止编码）  
> **参考基准**: nginx 1.27.x (mainline), https://github.com/nginx/nginx

---

## 1. 执行摘要

本计划基于对 nginx 最新主线源码的完整目录结构、模块清单和构建系统的深度研究，提出将 Nginx 从 C 语言迁移到 Go 语言的系统性方案。核心策略是：**分层对等映射，而非逐行翻译** —— 保留 nginx 的架构精髓（模块化、事件驱动、分层配置），用 Go 的惯用模式（goroutine、channel、interface、标准库）重新实现。

---

## 2. Nginx 源码结构全景分析

### 2.1 目录结构总览

```
src/
├── core/          # 内核：内存池、哈希、红黑树、链表、字符串、日志、配置文件解析器
├── event/         # 事件系统：epoll/kqueue/poll/select + 定时器 + 连接管理
│   ├── modules/   # 平台特定事件驱动：epoll(Linux), kqueue(BSD), select/poll(通用)
│   └── quic/      # QUIC 协议 + BPF 加速
├── http/          # HTTP 协议栈
│   ├── modules/   # 70+ HTTP 模块（filter + handler + upstream）
│   ├── v2/        # HTTP/2 实现
│   ├── v3/        # HTTP/3 (QUIC) 实现
│   └── perl/      # Perl 嵌入模块
├── stream/        # TCP/UDP 四层代理（L4 load balancer）
├── mail/          # IMAP/POP3/SMTP 代理
├── os/            # 操作系统适配层：unix / win32
└── misc/          # 杂项：Google Perftools, C++ 测试模块
```

### 2.2 核心架构模式

| 机制 | C 实现方式 | Go 迁移思路 |
|------|-----------|------------|
| **内存管理** | `ngx_palloc` 内存池（arena + slab） | `sync.Pool` + 对象复用；Go GC 替代手动释放 |
| **并发模型** | 多进程（master + worker）+ 单线程事件循环 | 单进程多 goroutine；Go runtime scheduler 替代进程模型 |
| **事件驱动** | `epoll_wait`/`kevent` + 回调函数 | Go `net` poller（内部 epoll/kqueue）+ goroutine per conn；或 `cloudwego/netpoll` 做零拷贝优化 |
| **配置系统** | 自定义 `ngx_conf_file` 解析器，模块注册 command | Go 结构体 + `flag`/`yaml`/`toml` 或自研兼容 nginx 配置语法的解析器 |
| **模块化** | `ngx_module_t` 结构体 + 编译期链接 | Go `interface` + 插件化（`hashicorp/go-plugin` 或 Go 原生 plugin） |
| **请求处理链** | 双向链表 `ngx_http_phase_engine` + 阶段处理器 | Go `handler` chain / `middleware` 模式（类似 Gin/Echo） |
| **upstream** | 共享内存 + 权重轮询/哈希/最小连接 | Go 内置负载均衡策略 + 健康检查 goroutine |
| **filter** | 链表式 `ngx_chain_t` 输出过滤 | Go `io.Reader`/`io.Writer` 管道 + `bufio` 缓冲 |
| **共享内存** | `ngx_shm` + `ngx_slab` + 原子锁 | 同一进程内无需共享内存；如需跨进程用外部存储（etcd/redis） |

---

## 3. 模块分类清单：Migrate / Skip / Defer

### 3.1 分类标准

- **Migrate**: 核心功能，必须保留，优先实现
- **Skip**: 已废弃、功能重复、或 Go 生态已有更好替代，明确不实现
- **Defer**: 非核心功能，或需要依赖其他模块，延后实现

### 3.2 HTTP 模块分类

| 模块 | 状态 | 说明 |
|------|------|------|
| **ngx_http_core_module** | **Migrate** | HTTP 核心，虚拟主机、location 匹配、阶段引擎 |
| **ngx_http_log_module** | **Migrate** | Access log，可扩展为结构化日志（JSON） |
| **ngx_http_upstream_module** | **Migrate** | 反向代理核心，负载均衡框架 |
| **ngx_http_static_module** | **Migrate** | 静态文件服务，Go `http.FileServer` 基础 + 优化 |
| **ngx_http_index_module** | **Migrate** | 目录索引文件 |
| **ngx_http_proxy_module** | **Migrate** | HTTP 反向代理，最核心的生产功能 |
| **ngx_http_fastcgi_module** | **Defer** | FastCGI 协议（PHP 等），协议较旧但仍有需求 |
| **ngx_http_uwsgi_module** | **Defer** | uWSGI 协议，Python 生态 |
| **ngx_http_scgi_module** | **Defer** | SCGI 协议，较少使用 |
| **ngx_http_grpc_module** | **Migrate** | gRPC 代理，现代微服务核心 |
| **ngx_http_ssl_module** | **Migrate** | TLS/SSL，Go `crypto/tls` 替代 OpenSSL |
| **ngx_http_v2_module** | **Migrate** | HTTP/2，Go `net/http` 内置支持 |
| **ngx_http_v3_module** | **Defer** | HTTP/3/QUIC，Go 生态不成熟（quic-go 可选） |
| **ngx_http_gzip_filter_module** | **Migrate** | Gzip 压缩，Go `compress/gzip` |
| **ngx_http_gunzip_filter_module** | **Defer** | Gunzip 解压，较少用 |
| **ngx_http_gzip_static_module** | **Defer** | 预压缩静态文件 |
| **ngx_http_chunked_filter_module** | **Migrate** | Chunked transfer，Go `net/http` 内置 |
| **ngx_http_range_filter_module** | **Migrate** | Range 请求处理 |
| **ngx_http_headers_filter_module** | **Migrate** | 自定义响应头 |
| **ngx_http_rewrite_module** | **Migrate** | URL 重写，正则表达式规则 |
| **ngx_http_access_module** | **Migrate** | IP 访问控制 |
| **ngx_http_limit_conn_module** | **Migrate** | 连接数限制 |
| **ngx_http_limit_req_module** | **Migrate** | 请求速率限制（漏桶算法） |
| **ngx_http_auth_basic_module** | **Migrate** | HTTP Basic 认证 |
| **ngx_http_auth_request_module** | **Migrate** | 子请求认证 |
| **ngx_http_geo_module** | **Migrate** | 基于 IP 的变量映射 |
| **ngx_http_geoip_module** | **Skip** | 依赖 MaxMind GeoIP 旧库；Go 有 `oschwald/geoip2-golang` |
| **ngx_http_map_module** | **Migrate** | 变量映射 |
| **ngx_http_split_clients_module** | **Migrate** | A/B 测试分流 |
| **ngx_http_referer_module** | **Skip** | 防盗链，功能简单且可用其他方式实现 |
| **ngx_http_realip_module** | **Migrate** | 从 X-Forwarded-For 获取真实 IP |
| **ngx_http_browser_module** | **Skip** | 浏览器检测，已过时（User-Agent 检测不再可靠） |
| **ngx_http_charset_filter_module** | **Defer** | 字符集转换，Go 原生 `unicode/utf8` 覆盖大部分场景 |
| **ngx_http_addition_filter_module** | **Defer** | 在响应前后添加内容 |
| **ngx_http_sub_filter_module** | **Defer** | 响应内容替换 |
| **ngx_http_ssi_filter_module** | **Skip** | Server Side Includes，过时技术，前端构建工具替代 |
| **ngx_http_userid_filter_module** | **Skip** | Cookie 用户跟踪，隐私法规下不推荐 |
| **ngx_http_dav_module** | **Defer** | WebDAV，小众需求 |
| **ngx_http_autoindex_module** | **Migrate** | 目录列表，Go `http.FileServer` 可实现 |
| **ngx_http_random_index_module** | **Defer** | 随机目录索引，边缘功能 |
| **ngx_http_empty_gif_module** | **Skip** | 1x1 透明 GIF，过时常用技巧 |
| **ngx_http_flv_module** | **Skip** | FLV 伪流媒体，Flash 已死 |
| **ngx_http_mp4_module** | **Defer** | MP4 伪流媒体，现代用 HLS/DASH |
| **ngx_http_secure_link_module** | **Defer** | 安全链接校验，可用 JWT 替代 |
| **ngx_http_degradation_module** | **Skip** | 内存不足时降级，Go 有内置 GC 和 OOM 处理 |
| **ngx_http_stub_status_module** | **Migrate** | 状态监控页，需 Prometheus/metrics 增强 |
| **ngx_http_mirror_module** | **Defer** | 流量镜像，生产诊断用途 |
| **ngx_http_try_files_module** | **Migrate** | 文件存在性检查 |
| **ngx_http_perl_module** | **Skip** | Perl 嵌入，Go 不支持；可用 WASM/插件替代 |
| **ngx_http_xslt_filter_module** | **Skip** | XSLT 转换，过时技术 |
| **ngx_http_image_filter_module** | **Skip** | 图像处理（裁剪/缩放），用外部服务（如 imgproxy） |
| **ngx_http_tunnel_module** | **Defer** | CONNECT 隧道代理 |
| **ngx_http_proxy_v2_module** | **Defer** | PROXY protocol v2 |
| **ngx_http_memcached_module** | **Skip** | Memcached 代理，用原生 Go 客户端 |
| **ngx_http_upstream_hash_module** | **Migrate** | 一致性哈希负载均衡 |
| **ngx_http_upstream_ip_hash_module** | **Migrate** | IP 哈希 |
| **ngx_http_upstream_least_conn_module** | **Migrate** | 最小连接数 |
| **ngx_http_upstream_least_time_module** | **Migrate** | 最小响应时间（商业版） |
| **ngx_http_upstream_random_module** | **Migrate** | 随机选择 |
| **ngx_http_upstream_keepalive_module** | **Migrate** | 连接池 keepalive |
| **ngx_http_upstream_zone_module** | **Migrate** | upstream 共享内存配置 |
| **ngx_http_upstream_sticky_module** | **Defer** | Session 粘性（商业版） |
| **ngx_http_slice_filter_module** | **Defer** | 大文件分段缓存 |
| **ngx_http_not_modified_filter_module** | **Migrate** | 304 缓存处理 |
| **ngx_http_copy_filter_module** | **Migrate** | 子请求响应拷贝 |
| **ngx_http_postpone_filter_module** | **Defer** | 子请求输出排序 |
| **ngx_http_header_filter_module** | **Migrate** | 默认响应头生成 |
| **ngx_http_write_filter_module** | **Migrate** | 输出缓冲写入 |
| **ngx_http_range_header_filter_module** | **Migrate** | Range 响应头处理 |
| **ngx_http_v2_filter_module** | **Migrate** | HTTP/2 响应帧 |
| **ngx_http_v3_filter_module** | **Skip** | HTTP/3 响应帧 — 已确认不纳入路线图 |
| **ngx_http_status_module** | **Skip** | 已废弃（`--without-http_upstream_sticky` 废弃时关联） |

### 3.3 Stream (L4) 模块分类

| 模块 | 状态 | 说明 |
|------|------|------|
| **ngx_stream_core_module** | **Migrate** | TCP/UDP 代理核心 |
| **ngx_stream_proxy_module** | **Migrate** | 四层代理转发 |
| **ngx_stream_ssl_module** | **Migrate** | Stream TLS |
| **ngx_stream_ssl_preread_module** | **Migrate** | SNI 路由（不终止 TLS） |
| **ngx_stream_upstream_*_module** | **Migrate** | 负载均衡策略（round_robin, hash, least_conn, random） |
| **ngx_stream_return_module** | **Defer** | 返回固定内容 |
| **ngx_stream_pass_module** | **Migrate** | 直接转发 |
| **ngx_stream_access_module** | **Migrate** | IP 访问控制 |
| **ngx_stream_limit_conn_module** | **Migrate** | 连接限制 |
| **ngx_stream_realip_module** | **Migrate** | 真实 IP |
| **ngx_stream_geo_module** | **Migrate** | IP 地理映射 |
| **ngx_stream_geoip_module** | **Skip** | 同 HTTP geoip |
| **ngx_stream_map_module** | **Migrate** | 变量映射 |
| **ngx_stream_split_clients_module** | **Migrate** | A/B 分流 |
| **ngx_stream_set_module** | **Migrate** | 变量设置 |

### 3.4 Mail 模块分类

| 模块 | 状态 | 说明 |
|------|------|------|
| **全部 Mail 模块** | **Skip** | IMAP/POP3/SMTP 代理 — 已确认现阶段直接 Skip，Go 生态有更好专用方案（如 Haraka, Stalwart） |

### 3.5 核心/事件模块分类

| 模块 | 状态 | 说明 |
|------|------|------|
| **ngx_epoll_module** | **Skip** | Go runtime 内部已使用 epoll/kqueue（通过 `netpoller`） |
| **ngx_kqueue_module** | **Skip** | 同上 |
| **ngx_poll_module** | **Skip** | 同上 |
| **ngx_select_module** | **Skip** | 同上 |
| **ngx_devpoll_module** | **Skip** | Solaris，Go 不支持 |
| **ngx_eventport_module** | **Skip** | Solaris，Go 不支持 |
| **ngx_iocp_module** | **Skip** | Windows IOCP，Go 在 Windows 用不同机制 |
| **ngx_win32_*_module** | **Skip** | Windows 特定，Go 跨平台抽象已覆盖 |
| **ngx_thread_pool** | **Skip** | Go goroutine 替代线程池 |
| **ngx_palloc** | **Skip** | Go GC + `sync.Pool` 替代 |
| **ngx_slab** | **Skip** | 共享内存 slab，Go 单进程无需 |
| **ngx_shmtx** | **Skip** | 共享内存锁，Go 单进程无需 |
| **ngx_resolver** | **Migrate** | DNS 解析，Go `net.Resolver` 增强版 |
| **ngx_open_file_cache** | **Migrate** | 文件描述符缓存，Go 可用或 OS 自带 |
| **ngx_regex** | **Skip** | Go `regexp` 包替代 |
| **ngx_crypt** | **Skip** | Go `crypto` 包替代 |

### 3.6 统计摘要

| 类别 | 模块数量 | Migrate | Defer | Skip |
|------|---------|---------|-------|------|
| HTTP 核心 + Filter | 47 | 32 | 9 | 6 |
| HTTP 处理器 + Upstream | 30 | 20 | 7 | 3 |
| Stream | 20 | 14 | 2 | 4 |
| Mail | 12 | 0 | 0 | 12 |
| 核心/事件 | 15 | 2 | 0 | 13 |
| **总计** | **124** | **68 (55%)** | **18 (15%)** | **38 (31%)** |

---

## 4. Go 架构替代方案设计

### 4.1 网络模型选型

**方案 A: 标准库 `net/http` + `net`（推荐作为基线，已确认）**

```go
// 基线实现：利用 Go 原生网络栈
server := &http.Server{
    Addr:         ":8080",
    Handler:      router,
    ReadTimeout:  60 * time.Second,
    WriteTimeout: 60 * time.Second,
    IdleTimeout:  120 * time.Second,
}
```

- **优点**: 稳定、生态丰富、HTTP/2 内置、TLS 内置、跨平台、无已知安全漏洞
- **缺点**: 每个连接一个 goroutine（内存 ~2-4KB/conn），C10M 场景有瓶颈
- **适用**: 80% 的场景，开发速度优先，符合 gonx 第一优先级

**方案 B: `cloudwego/netpoll`（性能优先场景，需明确优势证明）**

```go
// 零拷贝 + 事件驱动 + goroutine 池
import "github.com/cloudwego/netpoll"

listener, _ := netpoll.CreateListener("tcp", ":8080")
eventLoop, _ := netpoll.NewEventLoop(handler)
eventLoop.Serve(listener)
```

- **优点**: 字节跳动生产验证（Kitex/Hertz），减少 goroutine 数量（1:1 goroutine:request vs 1:1 goroutine:conn），零拷贝读写，适合 RPC/高并发
- **缺点**: 不支持 Windows；API 不兼容 `net/http`；学习曲线；阻塞粒度从 g 上升到 m（与 Go GMP 调度哲学冲突）
- **安全风险**: 间接依赖 CVE-2022-29526（中危，golang.org/x/sys）和 CVE-2022-41723（高危，golang.org/x/net），需保持依赖更新
- **安全记录**: 无直接发布的 security advisories；无 SECURITY.md；建议谨慎评估
- **性能数据**: 官方 benchmark 显示 QPS 提升 ~184%（128K vs 45K），内存减少 ~71%，延迟降低 ~68%（RPC 场景，1000 并发，4核8G）
- **适用**: 仅在 benchmark 验证有明确优势后，作为可选编译标签接入

**方案 C: `gnet`（纯事件驱动）**

- **优点**: 类似 Redis 的单线程事件循环，延迟极低
- **缺点**: Go 1.23+ 的 netpoller 已大幅优化，收益递减；需自行处理 HTTP 协议
- **适用**: 特定定制协议场景，gonx 暂不考虑

### 4.2 对象池设计（替代 `ngx_palloc`）

```go
package pool

import "sync"

// BufferPool 替代 nginx 的 ngx_chain_t / buf 复用
type BufferPool struct {
    pool sync.Pool
}

func NewBufferPool(size int) *BufferPool {
    return &BufferPool{
        pool: sync.Pool{
            New: func() interface{} {
                return make([]byte, size)
            },
        },
    }
}

func (p *BufferPool) Get() []byte {
    return p.pool.Get().([]byte)
}

func (p *BufferPool) Put(b []byte) {
    if cap(b) == 0 { return }
    p.pool.Put(b[:cap(b)])
}

// RequestPool 替代 ngx_pool_t per-request allocation
type RequestPool struct {
    pool sync.Pool
}

// 请求结束后 Reset 回池，无需手动 free
```

**关键决策**: Go 的 GC 已足够高效（1.24+ 分代 GC），`sync.Pool` 仅用于高频分配对象（buffer、header slice）。不要试图重建 nginx 的 arena 内存池——那是为了 C 的手动内存管理，Go 不需要。

### 4.3 配置解析器设计（替代 `ngx_conf_file`）

**推荐方案 A: 兼容 nginx 配置语法子集（已确认）**

自研递归下降解析器，支持 nginx 配置语法子集：

```nginx
http {
    server {
        listen 80;
        server_name example.com;
        location / {
            proxy_pass http://backend;
        }
    }
    upstream backend {
        server 127.0.0.1:8080 weight=5;
        server 127.0.0.1:8081;
        keepalive 32;
    }
}
```

解析器产出 Go 结构体：

```go
type Config struct {
    HTTP   HTTPConfig
    Stream *StreamConfig // optional
}

type HTTPConfig struct {
    Servers   []ServerConfig
    Upstreams map[string]UpstreamConfig
}

type ServerConfig struct {
    Listen      []ListenConfig
    ServerNames []string
    Locations   []LocationConfig
}
```

**方案 B/C: YAML/TOML（已排除）**
- 不纳入支持范围，保持配置语法单一性

**推荐策略（已确认）**:  
- **配置文件语法**: 仅支持 nginx 语法子集（兼容存量配置），内部转换为 Go 结构体  
- **动态配置**: 通过 Admin API (REST/gRPC) 修改运行时配置，无需 reload  
- **配置热重载**: 文件变更监听 + 差分更新（无需 worker 进程重启）

### 4.4 模块化架构（替代 `ngx_module_t`）

```go
package module

// Module 接口替代 ngx_module_t
type Module interface {
    Name() string
    Type() ModuleType // CORE, HTTP, STREAM, FILTER
    Commands() []Command
    Init(cycle *Cycle) error
    Exit(cycle *Cycle) error
}

type HTTPModule interface {
    Module
    // 阶段注册器，替代 nginx 的 phase handler
    RegisterHandlers(engine *HTTPPhaseEngine)
}

type HTTPPhaseEngine struct {
    // 同 nginx 的 11 个阶段，但用 slice 存储 handler
    PostRead      []PostReadHandler
    ServerRewrite []ServerRewriteHandler
    FindConfig    []FindConfigHandler
    Rewrite       []RewriteHandler
    PostRewrite   []PostRewriteHandler
    Preaccess     []PreaccessHandler
    Access        []AccessHandler
    PostAccess    []PostAccessHandler
    Precontent    []PrecontentHandler
    Content       []ContentHandler
    Log           []LogHandler
}
```

**Filter 链**（替代 ngx_chain_t 过滤）：

```go
type FilterFunc func(w *ResponseWriter, r *Request) error

type FilterChain struct {
    filters []FilterFunc
    index   int
}

func (fc *FilterChain) Next(w *ResponseWriter, r *Request) error {
    if fc.index >= len(fc.filters) {
        return nil
    }
    f := fc.filters[fc.index]
    fc.index++
    return f(w, r)
}
```

### 4.5 Upstream 负载均衡设计

```go
package upstream

// Peer 代表后端服务器
type Peer struct {
    Address    string
    Weight     int
    MaxFails   int
    FailTimeout time.Duration
    CurrentConns int64 // atomic
    CurrentFails int64 // atomic
    Healthy    atomic.Bool
}

// LoadBalancer 接口
type LoadBalancer interface {
    Select(peers []*Peer, req *http.Request) (*Peer, error)
    Name() string
}

// 实现：RoundRobin, LeastConn, IPHash, Hash, Random, ConsistentHash
```

**健康检查**: 独立 goroutine 定时探测（HTTP / TCP），结果写入 atomic bool，无需共享内存锁。

### 4.6 进程模型（替代 master/worker）

Go 不需要 nginx 的多进程模型：

| nginx 功能 | Go 替代方案 |
|-----------|------------|
| Master 进程管理 | 单进程，OS 通过 systemd/supervisor 管理 |
| Worker 进程 | goroutine pool，由 Go scheduler 调度到 GOMAXPROCS 线程 |
| 优雅重启 (graceful reload) | 监听新端口 + 连接迁移，或 systemd socket activation |
| 信号处理 (HUP/USR1/USR2) | `os/signal` + Admin API |
| CPU 亲和性 | `runtime.GOMAXPROCS(n)` + `taskset` |

---

## 5. 分层迁移路线图

### 5.1 架构分层映射

```
┌─────────────────────────────────────────────────────────────┐
│  Nginx Layer                    │  Gonx (Go) Layer          │
├─────────────────────────────────────────────────────────────┤
│  Master Process                 │  Go Main + OS Supervisor  │
│  Worker Process                 │  Goroutine Scheduler      │
│  Event Loop (epoll/kqueue)      │  Go Netpoller / netpoll   │
│  Connection Pool                │  net.Conn + sync.Pool     │
│  Memory Pool (ngx_palloc)       │  Go GC + sync.Pool        │
│  HTTP State Machine             │  HTTP Phase Engine        │
│  Location Router                │  Trie / Radix Tree Router │
│  Upstream LB                    │  LB Interface + Goroutines│
│  Filter Chain                   │  io.Pipe + Middleware     │
│  Configuration                  │  Parser → Go Struct → API │
│  Logging                        │  slog / zap / structured  │
│  Shared Memory (slab)           │  In-memory / External KV  │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 迁移阶段规划

| 阶段 | 目标 | 预计 PR 数 | 关键模块 |
|------|------|-----------|---------|
| **Phase 0: 骨架** | 项目结构、Makefile、CI、配置解析器基线 | PR-1~3 | `cmd/`, `pkg/config/`, `Makefile` |
| **Phase 1: HTTP 核心** | HTTP server、router、phase engine、静态文件 | PR-4~8 | `pkg/http/core`, `pkg/http/router`, `pkg/http/static` |
| **Phase 2: 反向代理** | Proxy、upstream、负载均衡、keepalive | PR-9~15 | `pkg/http/proxy`, `pkg/upstream/` |
| **Phase 3: 协议扩展** | HTTP/2, TLS/SSL, gRPC, WebSocket | PR-16~20 | `pkg/http/v2`, `pkg/tls`, `pkg/grpc` |
| **Phase 4: 过滤与功能** | Gzip, rewrite, access, rate limit, realip | PR-21~28 | `pkg/http/filters/`, `pkg/http/modules/` |
| **Phase 5: Stream (L4)** | TCP/UDP proxy, stream upstream, SNI | PR-29~35 | `pkg/stream/` |
| **Phase 6: 管理面** | Admin API, metrics, hot reload, health check | PR-36~42 | `pkg/admin/`, `pkg/metrics/` |
| **Phase 7: 优化** | 性能调优、netpoll 接入、零拷贝、缓存 | PR-43~50 | `pkg/perf/`, `pkg/cache/` |

### 5.3 与 gomq 的协同设计

考虑到用户同时主导 gomq（RabbitMQ → Go 迁移）项目，建议共享以下基础设施：

| 共享组件 | 位置 | 说明 |
|---------|------|------|
| 配置解析器 | `github.com/qdongxu/gonx/pkg/config` | 同 gomq 的配置风格一致 |
| 日志库 | `github.com/qdongxu/gonx/pkg/log` | 结构化 JSON 日志，同 gomq |
| Metrics | `github.com/qdongxu/gonx/pkg/metrics` | Prometheus 指标，同 gomq |
| Admin API 风格 | REST + gRPC | 同 gomq 的 Web 管理端风格一致 |
| 构建流程 | `Makefile` + `go:embed` | 同 gomq 的构建规范 |
| 测试规范 | `go test ./...` + 行宽 80 | 同 gomq 的硬性约束 |

---

## 6. 已废弃 / 遗产模块说明

以下模块在 nginx 源码中已被标记废弃或属于旧技术，明确 **Skip**：

| 模块/功能 | 废弃标记 | 替代方案 |
|-----------|---------|---------|
| `--with-ipv6` | 选项已废弃 | Go `net` 原生 IPv6 |
| `--with-imap` | 选项已废弃 | 专用邮件代理 |
| `--with-imap_ssl_module` | 选项已废弃 | 同上 |
| `--with-md5` / `--with-sha1` | 选项已废弃 | Go `crypto` |
| `ngx_http_browser_module` | 无直接标记 | 过时，Skip |
| `ngx_http_geoip_module` | 依赖旧库 | `geoip2-golang` |
| `ngx_http_ssi_module` | 无直接标记 | 过时，Skip |
| `ngx_http_flv_module` | 无直接标记 | Flash 已死，Skip |
| `ngx_http_perl_module` | 无直接标记 | Go 不支持 Perl 嵌入 |
| `ngx_http_empty_gif_module` | 无直接标记 | 过时常用技巧，Skip |
| `ngx_http_image_filter_module` | 无直接标记 | 外部服务替代 |
| `ngx_http_xslt_filter_module` | 无直接标记 | 过时，Skip |
| `ngx_http_degradation_module` | 无直接标记 | Go 内存管理替代 |
| `ngx_http_userid_filter_module` | 无直接标记 | 隐私法规不推荐 |
| `ngx_http_referer_module` | 无直接标记 | 简单功能可替代 |
| **Mail 全部模块** | — | 专用方案替代 |

---

## 7. 风险与注意事项

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| **Go GC 停顿** | 高并发下 GC 可能影响延迟 | 使用 Go 1.24+ 分代 GC；优化对象分配；必要时使用 `sync.Pool` |
| **HTTP/3 不成熟** | 生态依赖 `quic-go`，稳定性待验证 | Phase 3 后再评估；先用 HTTP/2 |
| **配置兼容性** | 存量 nginx 配置无法 100% 兼容 | 明确支持子集；提供迁移工具；文档标注差异 |
| **Filter 链性能** | Go 的 `io.Pipe` 可能有拷贝开销 | 关键路径用 `netpoll` 零拷贝；benchmark 验证 |
| **Windows 支持** | `netpoll` 不支持 Windows | 标准库作为 fallback；Windows 用标准库 |
| **调试复杂度** | Go 的 goroutine 调试比 C 单线程复杂 | pprof + trace + 结构化日志全覆盖 |
| **模块生态** | nginx 第三方模块生态（Lua, ModSecurity 等） | 定义清晰插件接口；优先原生实现；Lua 用 gopher-lua |

---

## 8. 参考来源

- **Nginx 源码**: https://github.com/nginx/nginx (mainline, 2026-05-30 clone)
- **Nginx 文档**: https://nginx.org/en/docs/
- **Nginx 模块构建脚本**: `auto/modules` (源码内)
- **Nginx 配置选项**: `auto/options` (源码内)
- **Go Netpoller**: https://dzone.com/articles/go-servers-understanding-epoll-kqueue-netpoll
- **CloudWeGo Netpoll**: https://github.com/cloudwego/netpoll
- **Go HTTP/2**: https://pkg.go.dev/golang.org/x/net/http2
- **Go Runtime Scheduler**: https://go.dev/src/runtime/proc.go

---

## 9. 已确认决策（2026-05-30）

| 决策项 | 确认结果 | 说明 |
|--------|---------|------|
| **配置语法** | ✅ 仅兼容 nginx 子集 | 不支持 YAML/TOML/双语法，保持单一性 |
| **网络层选型** | ✅ 标准库基线 | Phase 1~2 用 `net/http`；netpoll 仅在 benchmark 验证有明确优势后作为可选编译标签接入 |
| **HTTP/3** | ❌ 不纳入路线图 | 暂不实现，等 Go 生态成熟后评估 |
| **Mail 代理** | ❌ 直接 Skip | 现阶段不实现，Go 生态有专用方案 |
| **Lua 支持** | ❌ 不需要 | 不引入 gopher-lua |
| **WASM 插件** | ❌ 不需要 | 不引入 WASM 跨语言方案 |

## 10. 待决策事项

1. **netpoll 接入时机**: 是否需要 Phase 1 就设计 `netpoll` 兼容接口，还是 Phase 3 再考虑？
2. **Stream (L4) 优先级**: 是否需要在 HTTP 稳定前就启动 Stream 模块？
3. **Admin API 风格**: REST-only 还是 REST + gRPC 双协议？

---

> **本计划已更新为确认版本。下一阶段进入 Phase 0 骨架搭建。**
