// config.go
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var MainConfig = viper.New()
var EnvPrefix = ""

func Init() {
	MainConfig.SetEnvPrefix(EnvPrefix)
	MainConfig.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	MainConfig.AutomaticEnv()

	MainConfig.SetConfigFile(".env")
	if err := MainConfig.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting vars from env file: %+v\n", err)
	}
}
