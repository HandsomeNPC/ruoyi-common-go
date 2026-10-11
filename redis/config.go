package redis

// RedisConfig Redis 缓存/会话/分布式锁配置，对应 yaml 的 redis 段。
// 默认值与校验在 newRedisClient 构造客户端时统一兜底，host/port/db 缺失在启动期即报错。
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`

	ClientName      string `mapstructure:"clientName"`      // 连接标识
	PoolSize        int    `mapstructure:"poolSize"`        // 连接池大小，缺省 go-redis 默认(10×CPU)
	MinIdleConns    int    `mapstructure:"minIdleConns"`    // 最小空闲连接
	DialTimeoutMs   int    `mapstructure:"dialTimeoutMs"`   // 建连超时(毫秒)，缺省 5000
	ReadTimeoutMs   int    `mapstructure:"readTimeoutMs"`   // 读超时(毫秒)，缺省 3000
	WriteTimeoutMs  int    `mapstructure:"writeTimeoutMs"`  // 写超时(毫秒)，缺省 3000
	ConnMaxIdleTime int    `mapstructure:"connMaxIdleTime"` // 空闲连接最大存活(秒)
}
