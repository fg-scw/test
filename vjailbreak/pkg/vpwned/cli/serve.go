package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/platform9/vjailbreak/pkg/vpwned/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serverCmd = &cobra.Command{
	Use:   "serve",
	Short: "start the vpwctl server",
	Long:  "start the vpwctl server",
	Run: func(cmd *cobra.Command, args []string) {
		if val, err := cmd.Flags().GetString("host"); err == nil {
			Config.Host = val
		}
		if val, err := cmd.Flags().GetString("port"); err == nil {
			Config.Port = val
		}
		if val, err := cmd.Flags().GetString("api_port"); err == nil {
			Config.APIPort = val
		}
		serve()
	},
}

func initCfg() {
	var homeDir string
	var err error
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		homeDir, err = homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		viper.AddConfigPath(homeDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".vpwctl.yaml")
	}
	viper.SetEnvPrefix("VPWCTL")
	viper.SetEnvKeyReplacer(strings.NewReplacer("/", "_", "-", "_"))
	viper.AutomaticEnv()
	err = viper.ReadInConfig()
	if err != nil {
		logrus.Error(err, viper.ConfigFileUsed())
	} else {
		logrus.Info("using defaults")
	}

	Config.ParseConfig()
	logrus.Debugf("Config Parsed as: %+v", Config)
}

func init() {
	rootCmd.AddCommand(serverCmd)
	cobra.OnInitialize(initCfg)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Default config file $HOME/.vpwctl.yaml")
	if err := viper.BindEnv("host"); err != nil {
		logrus.Error("error in reading host", err)
	}
	if err := viper.BindEnv("port"); err != nil {
		logrus.Error("error in reading port", err)
	}
	if err := viper.BindEnv("api_port"); err != nil {
		logrus.Error("error in reading api_port", err)
	}
	viper.SetDefault("host", "127.0.0.1")
	viper.SetDefault("port", "3000")
	viper.SetDefault("api_port", "3001")
	viper.SetDefault("api_host", "0.0.0.0")
	serverCmd.PersistentFlags().StringP("host", "i", "127.0.0.1", "Set the IP-Address to listen on")
	serverCmd.PersistentFlags().StringP("port", "g", "3000", "Set the gRPC port to open")
	serverCmd.PersistentFlags().StringP("api_port", "r", "3001", "Set the REST API port to open")
	serverCmd.PersistentFlags().StringP("api_host", "a", "0.0.0.0", "Set the REST API host to listen on")
	logrus.SetLevel(Config.logLevel)
}

func serve() {
	// Run the server in a goroutine
	go func() {
		logrus.Info(Config.Host, Config.Port, Config.APIPort)
		if err := server.StartServer(Config.Host, Config.Port, Config.APIPort, Config.APIHost); err != nil {
			return
		}
	}()
	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Block until we receive an interrupt signal
	<-stop

	// Initiating graceful shutdown
	logrus.Info("Shutting down server...")

	// Create a deadline for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	os.Exit(1)
}
