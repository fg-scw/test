package cli

import (
	"fmt"
	"os"
	"strings"

	_ "github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers/base"
	_ "github.com/platform9/vjailbreak/pkg/vpwned/sdk/providers/maas"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var Config config

type config struct {
	Host     string
	Port     string
	APIPort  string
	APIHost  string
	logLevel logrus.Level
}

func parseLogLevel(logLevel string) (logrus.Level, error) {
	switch strings.ToLower(logLevel) {
	case "trace":
		return logrus.TraceLevel, nil
	case "debug":
		return logrus.DebugLevel, nil
	case "info":
		return logrus.InfoLevel, nil
	case "warn", "warning":
		return logrus.WarnLevel, nil
	case "error":
		return logrus.ErrorLevel, nil
	case "fatal":
		return logrus.FatalLevel, nil
	case "panic":
		return logrus.PanicLevel, nil
	default:
		return logrus.InfoLevel, fmt.Errorf("invalid log level: %s", logLevel)
	}
}

func (c *config) ParseConfig() {
	c.Host = viper.GetString("host")        // default: localhost
	c.Port = viper.GetString("port")        //default: 3000
	c.APIPort = viper.GetString("api_port") //default: 3001
	c.APIHost = viper.GetString("api_host")
	logLevel := logrus.InfoLevel
	env_logLevel := os.Getenv("LOG_LEVEL")
	if env_logLevel != "" {
		currLevel, err := parseLogLevel(env_logLevel)
		if err != nil {
			logrus.Error(err)
		} else {
			logLevel = currLevel
		}
	} else {
		currLevel, err := parseLogLevel(viper.GetString("log_level"))
		if err != nil {
			logrus.Error(err)
		} else {
			logLevel = currLevel
		}
	}
	c.logLevel = logLevel
}

var rootCmd = &cobra.Command{
	Use:   "vpwctl",
	Short: "start the vpwctl server",
}

func Execute() error {
	logLevel := logrus.InfoLevel
	env_logLevel := os.Getenv("LOG_LEVEL")
	if env_logLevel != "" {
		currLevel, err := parseLogLevel(env_logLevel)
		if err != nil {
			logrus.Error(err)
		} else {
			logLevel = currLevel
		}
	}
	logrus.SetLevel(logLevel)
	return rootCmd.Execute()
}
