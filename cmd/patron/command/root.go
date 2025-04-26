package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "patron",
	Short: "Patron CLI provides a collection of processes and tools while working with Patron",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/config/patron.yaml)")
	if err := viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config")); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// rootCmd.AddCommand(cmd1)
	// rootCmd.AddCommand(cmd2)
}

func initConfig() {
	viper.SetConfigName("patron")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/patron/")
	viper.AddConfigPath("$HOME/.patron")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("$HOME")
		viper.SetConfigName(".patron")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file:", err)
	}
}
