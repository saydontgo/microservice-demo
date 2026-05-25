# 本地目标运行拓扑

## 1. 说明

当前机器不是 PRD 指定运行环境。本项目目标运行在 Windows 11 + WSL2 单机环境。

## 2. 推荐拓扑

```text
WSL2 Ubuntu
  |-- frontend static server : 8080
  |-- api gateway            : 8000
  |-- auth service           : 8101
  |-- user service           : 8102
  |-- product service        : 8103
  |-- order service          : 8104
  |-- payment service        : 8105
  |-- inventory service      : 8106
  |-- analytics service      : 8107
  |-- mysql                  : 3306
  |-- redis                  : 6379
```

## 3. Demo 简化方案

为了降低单机启动成本，可将后端服务合并为一个 Go 进程，内部按模块分包：

```text
api-gateway + services in one process : 8000
mysql                                : 3306
redis                                : 6379
frontend                             : 8080
```

文档仍保持微服务边界，便于后续拆分。

## 4. 资源估计

| 组件 | 资源建议 |
| --- | --- |
| MySQL | 1 CPU，1-2GB 内存 |
| Redis | 0.5 CPU，512MB 内存 |
| Go 后端 | 1-2 CPU，1GB 内存 |
| 前端静态服务 | 低资源占用 |

目标机器 48GB 内存足够运行该 Demo。
