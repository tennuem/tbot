package configs

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestYandexProxyConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
		args []string
		want string
	}{
		{name: "default"},
		{name: "environment", env: "http://ru-proxy:3128", want: "http://ru-proxy:3128"},
		{name: "flag", args: []string{"--yandex.proxy_url=https://ru-proxy:3129"}, want: "https://ru-proxy:3129"},
		{name: "flag overrides environment", env: "http://env-proxy:3128", args: []string{"--yandex.proxy_url=http://flag-proxy:3128"}, want: "http://flag-proxy:3128"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldFlags, oldArgs := pflag.CommandLine, os.Args
			viper.Reset()
			pflag.CommandLine = pflag.NewFlagSet("test", pflag.ContinueOnError)
			os.Args = append([]string{"tbot"}, tc.args...)
			t.Cleanup(func() {
				pflag.CommandLine, os.Args = oldFlags, oldArgs
				viper.Reset()
			})
			t.Setenv("TBOT_YANDEX_PROXY_URL", tc.env)
			cfg := NewConfig()
			require.NoError(t, cfg.Read())
			require.Equal(t, tc.want, cfg.Yandex.ProxyURL)
		})
	}
}

func TestConfigJSONOmitsProxyCredentials(t *testing.T) {
	cfg := NewConfig()
	cfg.Yandex.ProxyURL = "http://proxy-user:proxy-secret@ru-proxy:3128"
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NotContains(t, string(data), cfg.Yandex.ProxyURL)
	require.NotContains(t, string(data), "proxy-user")
	require.NotContains(t, string(data), "proxy-secret")
}
