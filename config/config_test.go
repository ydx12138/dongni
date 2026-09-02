package config

import "testing"

// TestMysqlConfigDSN 验证 MySQL 配置会生成正确的数据源名称；无参数；返回 DSN 格式校验结果。
func TestMysqlConfigDSN(t *testing.T) {
	cfg := MysqlConfig{Host: "127.0.0.1", Port: 3306, Db: "blog", User: "root", Password: "password"}
	if got, want := cfg.DSN(), "root:password@tcp(127.0.0.1:3306)/blog?charset=utf8mb4&parseTime=true"; got != want {
		t.Fatalf("expected DSN %q, got %q", want, got)
	}
}

// TestApplyEnvironmentOverrides 验证 Docker 环境变量会覆盖数据库与 Redis 配置；无参数；返回配置覆盖校验结果。
func TestApplyEnvironmentOverrides(t *testing.T) {
	t.Setenv("BLOG_MYSQL_HOST", "db")
	t.Setenv("BLOG_MYSQL_PORT", "3306")
	t.Setenv("BLOG_MYSQL_DATABASE", "blog")
	t.Setenv("BLOG_MYSQL_PASSWORD", "mysql-password")
	t.Setenv("BLOG_REDIS_HOST", "redis")
	t.Setenv("BLOG_REDIS_PASSWORD", "redis-password")

	cfg := &Config{}
	applyEnvironmentOverrides(cfg)

	if cfg.MysqlConfig.Host != "db" || cfg.MysqlConfig.Port != 3306 || cfg.MysqlConfig.Db != "blog" || cfg.MysqlConfig.Password != "mysql-password" {
		t.Fatalf("expected MySQL configuration from environment, got %#v", cfg.MysqlConfig)
	}
	if cfg.Redis.Host != "redis" || cfg.Redis.Password != "redis-password" {
		t.Fatalf("expected Redis configuration from environment, got %#v", cfg.Redis)
	}
}

// TestApplyEnvironmentOverridesUsesRootPassword 验证 Docker env_file 注入的 MySQL root 密码可作为后端连接密码；无参数；返回配置覆盖校验结果。
func TestApplyEnvironmentOverridesUsesRootPassword(t *testing.T) {
	t.Setenv("MYSQL_ROOT_PASSWORD", "mysql-root-password")

	cfg := &Config{}
	applyEnvironmentOverrides(cfg)

	if cfg.MysqlConfig.Password != "mysql-root-password" {
		t.Fatalf("expected MySQL root password fallback, got %q", cfg.MysqlConfig.Password)
	}
}
