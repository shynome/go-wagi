# Changelog

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
