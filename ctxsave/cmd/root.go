package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ctxsave",
	Short: "Cross-model context saver — carry project knowledge between AI sessions",
	Long: `ctxsave captures context from Cursor sessions, git history, and manual notes,
stores it locally, and generates compressed context prompts you can paste
into a new session with any AI model.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
