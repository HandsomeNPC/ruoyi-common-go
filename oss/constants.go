package oss

import "time"

// 对象存储相关的 Redis key 与缓存组常量。
//
// 从 ruoyi-core/constant 迁入，使 oss 包不再依赖 ruoyi-core，配合 pkg/cache 使用。
const (
	// OssDefaultConfigKey 当前生效的对象存储配置键，值是 sys_oss_config.config_key。
	// 与 CacheSysOssConfig 缓存组分工：那边存各配置的完整 JSON，这里只记用哪一个。
	// "global:" 前缀沿用原 constant.GlobalRedisKey 的全局 key 约定。
	OssDefaultConfigKey = "global:sys_oss:default_config"
)

// 缓存组名与 TTL。
const (
	// CacheSysOss 单条对象存储记录的缓存组。
	CacheSysOss = "sys_oss"
	// CacheSysOssConfig 对象存储配置的缓存组，按 configKey 存各配置的完整 JSON。
	CacheSysOssConfig = "sys_oss_config"

	CacheTTLSysOss = 30 * 24 * time.Hour
	// CacheTTLSysOssConfig 不过期（key 不带 #ttl 后缀）。
	// 写路径都带缓存维护（新增/修改回写、改键清旧键、删除失效），
	// 且它是上传下载的必经依赖，过期拿不到配置会直接让上传失败。
	CacheTTLSysOssConfig = 0
)
