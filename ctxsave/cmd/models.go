package cmd

import (
	"fmt"

	"ctxsave/internal/generate"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(modelsCmd)
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List supported target models",
	Run: func(cmd *cobra.Command, args []string) {
		models := generate.ListModels()

		fmt.Printf("%-10s %-22s %-12s %s\n", "KEY", "MODEL", "CONTEXT", "DESCRIPTION")
		fmt.Println("──────────────────────────────────────────────────────────────────────────")
		for _, m := range models {
			ctx := formatTokens(m.ContextLimit)
			fmt.Printf("%-10s %-22s %-12s %s\n", m.Key, m.Name, ctx, m.Description)
		}
	},
}

func formatTokens(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%dM", n/1000000)
	}
	return fmt.Sprintf("%dK", n/1000)
}
