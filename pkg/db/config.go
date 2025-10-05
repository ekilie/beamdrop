package db

type Config struct {
	Password string
}

func (Config) TableName() string {
	return "server_config"
}