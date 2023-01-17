package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type environ struct {
	AppAddress     string `env:"RUN_ADDRESS" envDefault:"localhost:8000"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"localhost:8080"`
	DatabaseDSN    string `env:"DATABASE_URI" envDefault:"postgres://postgres:postgres@localhost:5432/postgres"`
}

type Flags struct {
	appAddress     string
	databaseDSN    string
	accrualAddress string
}

type Config struct {
	AppAddress     string
	DatabaseDSN    string
	AccrualAddress string
}

func GetAppFlags() Flags {
	flags := Flags{}
	flag.StringVar(&flags.appAddress, "a", "", "Address of application, for example: 0.0.0.0:8000")
	flag.StringVar(&flags.accrualAddress, "r", "", "Accrual system address, for example: localhost:8080")
	flag.StringVar(&flags.databaseDSN, "d", "", "Database connect source, for example: postgres://username:password@localhost:5432/database_name")
	flag.Parse()
	return flags
}

func NewAppConf(flags Flags) (*Config, error) {
	var err error
	var cfg Config
	var envs environ
	// Разбираю переменные среды и проверяю значение тегов на значение по умолчанию
	if err = env.Parse(&envs, env.Options{}); err != nil {
		return nil, err
	}
	// Определяю адрес сервера
	if flags.appAddress != "" {
		cfg.AppAddress = flags.appAddress
	} else {
		cfg.AppAddress = envs.AppAddress
	}
	// Определяю адрес системы начисления баллов.
	if flags.accrualAddress != "" {
		cfg.AccrualAddress = flags.accrualAddress
	} else {
		cfg.AccrualAddress = envs.AccrualAddress
	}
	// Определяю подключение к БД
	if flags.databaseDSN != "" {
		cfg.DatabaseDSN = flags.databaseDSN
	} else {
		cfg.DatabaseDSN = envs.DatabaseDSN
	}

	return &cfg, err
}
