# Changelog

## [Unrelease] - 

### Break Change

- 移除 gojs 支持

## [0.3.0] - 2023-03-24

### Change

- upgrade to wazero 1.0
- 新版 gojs wasm 的内存需求为 20M, 提高内存限制

## [0.2.1] - 2023-02-22

### Fix

- 不使用固定的运行限制时间, 改而跟随网关设置的超时时间
- 编译 wasm 的时长限制为 1m

## [0.2.0] - 2023-02-20

### Change

- 添加内存(10M)和时间(10s)限制

## [0.1.3] - 2023-01-31

### Fix

- 不使用 Compiler 改用解释器, 冷启动更快；解释器不支持 cache, 移除 cache dir 选项

## [0.1.1] - 2023-01-30

### Improve

- 添加 `Version`

## [0.1.0] - 2023-01-29

### Add

- 添加 `/dev/tcp/127.0.0.1/5432` 网络文件访问支持(udp 也支持), 将环境变量 `WASI_NET` 设置为 `allow` 开启
- 支持 `wazero cachedir` 选项, 默认开启, 因为重新编译的时间太长了

# Changelog

## [0.0.2] - 2023-01-27

### Add

- 支持 cache
- 支持 GOWASM
