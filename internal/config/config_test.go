package config

import "testing"

func TestConfigMySQLDSN_BitsUT(t *testing.T) {
	cfg := Config{
		MySQLHost:     "mysql",
		MySQLPort:     "3306",
		MySQLDatabase: "microservice_demo",
		MySQLUser:     "demo",
		MySQLPassword: "secret",
	}

	got := cfg.MySQLDSN()
	want := "demo:secret@tcp(mysql:3306)/microservice_demo?parseTime=true&charset=utf8mb4&loc=Local"
	if got != want {
		t.Fatalf("MySQLDSN() = %q, want %q", got, want)
	}
}

func TestConfigRedisOptions_BitsUT(t *testing.T) {
	cfg := Config{
		RedisHost:     "redis",
		RedisPort:     "6379",
		RedisPassword: "pwd",
		RedisDB:       1,
	}

	got := cfg.RedisOptions()
	if got.Addr != "redis:6379" {
		t.Fatalf("Addr = %q, want redis:6379", got.Addr)
	}
	if got.Password != "pwd" {
		t.Fatalf("Password = %q, want pwd", got.Password)
	}
	if got.DB != 1 {
		t.Fatalf("DB = %d, want 1", got.DB)
	}
}
