package config

import (
	"encoding/json"
	"io"
	"os"
)

// Configuration of app
type Configuration struct {
	RznUrl           string `required:"true" split_words:"true"`
	YaUrl            string `required:"true" split_words:"true"`
	BotToken         string `required:"true" split_words:"true"`
	MainChannel      int64  `required:"true" split_words:"true"`
	ModeratorChannel int64  `required:"true" split_words:"true"`
}

// LoadConfig load configuration from file
func LoadConfig() (*Configuration, error) {
	var (
		jsonFile *os.File
		err      error
	)
	if jsonFile, err = os.Open("config.json"); err != nil {
		return nil, err
	}
	var b []byte
	if b, err = io.ReadAll(jsonFile); err != nil {
		return nil, err
	}
	var conf *Configuration
	if err = json.Unmarshal(b, &conf); err != nil {
		return nil, err
	}
	return conf, err
}
