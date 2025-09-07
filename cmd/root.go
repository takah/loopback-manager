package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/takah/loopback-manager/internal/config"
	"github.com/takah/loopback-manager/internal/manager"
)

var (
	cfgFile string
	mgr     *manager.Manager
)

var rootCmd = &cobra.Command{
	Use:   "loopback-manager",
	Short: "Manage loopback IP addresses for Docker Compose projects",
	Long: `A CLI tool to manage loopback IP addresses for GitHub organization repositories
with Docker Compose configurations. Automatically assigns unique IP addresses
and prevents conflicts.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		mgr = manager.New(cfg)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all IP assignments",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		if err := mgr.List(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var assignCmd = &cobra.Command{
	Use:   "assign <org> <repo>",
	Short: "Assign IP to repository",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		if err := mgr.Assign(args[0], args[1], ip); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <org> <repo>",
	Short: "Remove IP assignment",
	Aliases: []string{"rm", "del"},
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := mgr.Remove(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for unassigned repositories",
	Run: func(cmd *cobra.Command, args []string) {
		if err := mgr.Scan(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var autoAssignCmd = &cobra.Command{
	Use:   "auto-assign",
	Short: "Auto-assign IPs to all unassigned repositories",
	Aliases: []string{"auto"},
	Run: func(cmd *cobra.Command, args []string) {
		if err := mgr.AutoAssign(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for duplicate IPs",
	Aliases: []string{"validate"},
	Run: func(cmd *cobra.Command, args []string) {
		if err := mgr.CheckDuplicates(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/loopback-manager/config.yaml)")
	
	assignCmd.Flags().StringP("ip", "i", "", "Specific IP address to assign")
	
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(assignCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(autoAssignCmd)
	rootCmd.AddCommand(checkCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		viper.AddConfigPath(home + "/.config/loopback-manager")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
