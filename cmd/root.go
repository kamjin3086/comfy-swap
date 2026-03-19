package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagServer = "http://localhost:8189"
	flagQuiet  bool
	flagJSON   bool
	flagPretty bool
)

var rootCmd = &cobra.Command{
	Use:   "comfy-swap",
	Short: "comfy-swap CLI and server",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagServer, "server", "s", envOrDefault("COMFY_SWAP_URL", "http://localhost:8189"), "comfy-swap server URL")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "quiet output")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", true, "json output")
	rootCmd.PersistentFlags().BoolVar(&flagPretty, "pretty", false, "pretty output")
}

func printResult(v interface{}) error {
	if flagQuiet {
		switch x := v.(type) {
		case string:
			fmt.Println(x)
			return nil
		case []string:
			for _, s := range x {
				fmt.Println(s)
			}
			return nil
		}
	}
	enc := json.NewEncoder(os.Stdout)
	if flagPretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

func envOrDefault(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
