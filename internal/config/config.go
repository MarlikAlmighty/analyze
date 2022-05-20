package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Configuration of app
type Configuration struct {
	RedisUrl string `required:"true" split_words:"true"`
	RznUrl   string `required:"true" split_words:"true"`
	YaUrl    string `required:"true" split_words:"true"`
	BotToken string `required:"true" split_words:"true"`
	Channel  string
}

func New() *Configuration {
	return &Configuration{}
}

// GetEnv configuration init
func (cnf *Configuration) GetEnv() error {
	if err := envconfig.Process("", cnf); err != nil {
		return err
	}
	return nil
}
