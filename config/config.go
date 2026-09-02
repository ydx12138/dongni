package config

import (
	"blog/flags"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	Cfg *Config
)

// 总的结构体
type Config struct {
	//Server       ServerConfig `mapstructure:"server"`
	CORS         CORSConfig   `mapstructure:"cors"`
	SystemConfig SystemConfig `mapstructure:"system"`
	LogConfig    LogConfig    `mapstructure:"log"`
	MysqlConfig  MysqlConfig  `mapstructure:"mysql"`
	Redis        RedisConfig  `mapstructure:"redis"`
	OssConfig    OssConfig    `mapstructure:"oss"`
	MailConfig   MailConfig   `mapstructure:"mail"`
	Wechat       WechatConfig `mapstructure:"wechat"`
	Sms          SmsConfig    `mapstructure:"sms"`
}

type WechatConfig struct {
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}
type MailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	FromName string `mapstructure:"from_name"`
	SSL      bool   `mapstructure:"ssl"`
}

type MysqlConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Db        string `mapstructure:"db"`
	User      string `mapstructure:"user"`
	Password  string `mapstructure:"password"`
	Log_level string `mapstructure:"log_level"`
}

func (m MysqlConfig) DSN() string {
	return m.User + ":" + m.Password + "@tcp(" + m.Host + ":" + strconv.Itoa(m.Port) + ")/" + m.Db + "?charset=utf8mb4&parseTime=true"

}

type SystemConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Env  string `mapstructure:"env"`
}

func (s SystemConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type LogConfig struct {
	App   string `mapstructure:"app"`
	Dir   string `mapstructure:"dir"`
	Level string `mapstructure:"level"`
}
type OssConfig struct {
	AccessKeyId     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Endpoint        string `mapstructure:"endpoint"`
	Bucket          string `mapstructure:"bucket"`
	Image_path      string `mapstructure:"image_path"`
	DefaultAvatar   string `mapstructure:"default_avatar"`
}

/*type ServerConfig struct {
	Port int `mapstructure:"port"`
}*/

type CORSConfig struct {
	AllowOrigins     []string      `mapstructure:"allow_origins"`
	AllowMethods     []string      `mapstructure:"allow_methods"`
	AllowHeaders     []string      `mapstructure:"allow_headers"`
	AllowCredentials bool          `mapstructure:"allow_credentials"`
	MaxAge           time.Duration `mapstructure:"max_age"`
}

type SmsConfig struct {
	SchemeName       string `mapstructure:"scheme_name"`        //方案名称
	CountryCode      string `mapstructure:"country_code"`       //号码国家编码
	SignName         string `mapstructure:"sign_name"`          //签名名称
	TemplateCode     string `mapstructure:"template_code"`      //短信模板CODE
	TemplateParam    string `mapstructure:"template_param"`     //短信模板
	CodeLength       int64  `mapstructure:"code_length"`        //验证码长度
	ValidTime        int64  `mapstructure:"valid_time"`         //验证码有效时间
	DuplicatePolicy  int64  `mapstructure:"duplicate_policy"`   //核验规则
	Interval         int64  `mapstructure:"interval"`           //时间间隔
	CodeType         int64  `mapstructure:"code_type"`          //生成的验证码类型
	ReturnVerifyCode bool   `mapstructure:"return_verify_code"` //是否返回验证码
	AutoRetry        int64  `mapstructure:"auto_retry"`         //是否自动替换签名重试
	AccessKeyId      string `mapstructure:"access_key_id"`
	AccessKeySecret  string `mapstructure:"access_key_secret"`
	CaseAuthPolicy   int64  `mapstructure:"case_auth_policy"`
}

func LoadConfig() (*Config, error) {

	// 1. 加载 .env 文件到环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// 2. 初始化 Viper
	v := viper.New()
	v.SetConfigFile("settings.yaml")
	v.SetConfigType("yaml")

	// 3. 绑定环境变量（自动将 YAML 键名转换为环境变量）
	// 例如：server.port -> ENJOYMALL_SERVER_PORT
	v.SetEnvPrefix("BLOG")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.BindEnv("mysql.db", "BLOG_MYSQL_DATABASE", "MYSQL_DATABASE"); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := v.ReadInConfig(); err != nil {
		return cfg, err
	}
	if err := v.Unmarshal(cfg); err != nil {
		return cfg, err
	}
	applyEnvironmentOverrides(cfg)
	zap.L().Info("读取配置文件" + flags.FlagOptions.File + "成功")
	Cfg = cfg
	return cfg, nil
}

// applyEnvironmentOverrides 用 BLOG_ 环境变量覆盖容器运行时配置；参数 cfg 为已解析的 YAML 配置；返回值为空。
func applyEnvironmentOverrides(cfg *Config) {
	cfg.SystemConfig.Host = environmentString("BLOG_SYSTEM_HOST", cfg.SystemConfig.Host)
	cfg.SystemConfig.Port = environmentInt("BLOG_SYSTEM_PORT", cfg.SystemConfig.Port)
	cfg.SystemConfig.Env = environmentString("BLOG_SYSTEM_ENV", cfg.SystemConfig.Env)
	cfg.LogConfig.Level = environmentString("BLOG_LOG_LEVEL", cfg.LogConfig.Level)

	cfg.MysqlConfig.Host = environmentString("BLOG_MYSQL_HOST", cfg.MysqlConfig.Host)
	cfg.MysqlConfig.Port = environmentInt("BLOG_MYSQL_PORT", cfg.MysqlConfig.Port)
	cfg.MysqlConfig.Db = environmentString("BLOG_MYSQL_DATABASE", cfg.MysqlConfig.Db)
	cfg.MysqlConfig.User = environmentString("BLOG_MYSQL_USER", cfg.MysqlConfig.User)
	cfg.MysqlConfig.Password = environmentString("BLOG_MYSQL_PASSWORD", environmentString("MYSQL_ROOT_PASSWORD", cfg.MysqlConfig.Password))

	cfg.Redis.Host = environmentString("BLOG_REDIS_HOST", cfg.Redis.Host)
	cfg.Redis.Port = environmentInt("BLOG_REDIS_PORT", cfg.Redis.Port)
	cfg.Redis.Password = environmentString("BLOG_REDIS_PASSWORD", cfg.Redis.Password)

	cfg.OssConfig.AccessKeyId = environmentString("BLOG_OSS_ACCESS_KEY_ID", cfg.OssConfig.AccessKeyId)
	cfg.OssConfig.AccessKeySecret = environmentString("BLOG_OSS_ACCESS_KEY_SECRET", cfg.OssConfig.AccessKeySecret)
	cfg.OssConfig.Endpoint = environmentString("BLOG_OSS_ENDPOINT", cfg.OssConfig.Endpoint)
	cfg.OssConfig.Bucket = environmentString("BLOG_OSS_BUCKET", cfg.OssConfig.Bucket)
	cfg.OssConfig.DefaultAvatar = environmentString("BLOG_OSS_DEFAULT_AVATAR", cfg.OssConfig.DefaultAvatar)

	cfg.MailConfig.Username = environmentString("BLOG_MAIL_USERNAME", cfg.MailConfig.Username)
	cfg.MailConfig.Password = environmentString("BLOG_MAIL_PASSWORD", cfg.MailConfig.Password)

	cfg.Wechat.AppID = environmentString("BLOG_WECHAT_APP_ID", cfg.Wechat.AppID)
	cfg.Wechat.AppSecret = environmentString("BLOG_WECHAT_APP_SECRET", cfg.Wechat.AppSecret)
}

// environmentString 读取非空环境变量；参数 key 为变量名、fallback 为默认值；返回环境变量或默认值。
func environmentString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// environmentInt 读取正整数环境变量；参数 key 为变量名、fallback 为默认值；返回环境变量或默认值。
func environmentInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

//func Load() (*Config, error) {
//	// 1. 加载 .env 文件到环境变量
//	if err := godotenv.Load(); err != nil {
//		log.Println("Warning: .env file not found, using environment variables")
//	}
//
//	// 2. 初始化 Viper
//	v := viper.New()
//	v.SetConfigName("config")
//	v.SetConfigType("yaml")
//	v.AddConfigPath("./configs")
//	v.AddConfigPath(".")
//
//	// 3. 绑定环境变量（自动将 YAML 键名转换为环境变量）
//	// 例如：server.port -> ENJOYMALL_SERVER_PORT
//	v.SetEnvPrefix("ENJOYMALL")
//	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
//	v.AutomaticEnv()
//
//	// 4. 显式绑定 Coze 配置的环境变量（处理下划线键名）
//	v.BindEnv("coze.bot_id", "ENJOYMALL_COZE_BOT_ID")
//	v.BindEnv("coze.review_summary_bot_id", "ENJOYMALL_COZE_REVIEW_SUMMARY_BOT_ID")
//	v.BindEnv("coze.access_token", "ENJOYMALL_COZE_ACCESS_TOKEN")
//	v.BindEnv("coze.api_url", "ENJOYMALL_COZE_API_URL")
//	v.BindEnv("coze.timeout", "ENJOYMALL_COZE_TIMEOUT")
//
//	// 5. 读取配置文件
//	if err := v.ReadInConfig(); err != nil {
//		log.Fatal("Error reading config.yaml:", err)
//	}
//
//	// 6. 解析配置到结构体
//
//	if err := v.Unmarshal(&cfg); err != nil {
//		log.Fatal("Error unmarshaling config:", err)
//	}
//
//	return cfg, nil
//}
