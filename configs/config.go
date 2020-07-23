package configs

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var ServiceName = "tbot"

var options = []option{
	{"telegram.token", "string", "token", "telegram token"},
	{"spotify.client_id", "string", "cid", "spotify client id"},
	{"spotify.client_secret", "string", "csecret", "spotify secret"},
	{"mongodb.addr", "string", "mongodb://root:root@localhost:27017/?ssl=false", "mongo addr"},
	{"logger.level", "string", "info", "logger level"},
}

type Config struct {
	Telegram struct {
		Token string
	}
	Spotify struct {
		ClientID     string `mapstructure:"client_id"`
		ClientSecret string `mapstructure:"client_secret"`
	}
	MongoDB struct {
		Addr string
	}
	Logger struct {
		Level string
	}
}

type option struct {
	name        string
	typing      string
	value       interface{}
	description string
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Read() error {
	viper.SetEnvPrefix(ServiceName)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	for _, o := range options {
		switch o.typing {
		case "string":
			pflag.String(o.name, o.value.(string), o.description)
		case "int":
			pflag.Int(o.name, o.value.(int), o.description)
		case "bool":
			pflag.Bool(o.name, o.value.(bool), o.description)
		case "float64":
			pflag.Float64(o.name, o.value.(float64), o.description)
		default:
			viper.SetDefault(o.name, o.value)
		}
	}
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	viper.BindPFlags(pflag.CommandLine)
	pflag.Parse()
	if fileName := viper.GetString("config"); fileName != "" {
		viper.SetConfigName(fileName)
		viper.SetConfigType("toml")
		viper.AddConfigPath(".")

		if err := viper.ReadInConfig(); err != nil {
			return err
		}
	}
	if err := viper.Unmarshal(c); err != nil {
		return err
	}
	return nil
}

func (c *Config) Print() error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(b))
	return nil
}
