package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"ctxsave/internal/store"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize context tracking in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		dbFile := filepath.Join(dir, ".ctxsave", "context.db")
		if _, err := os.Stat(dbFile); err == nil {
			fmt.Println("Already initialized — .ctxsave/context.db exists")
			return nil
		}

		st, err := store.New(dir)
		if err != nil {
			return fmt.Errorf("init failed: %w", err)
		}
		st.Close()

		fmt.Printf("Initialized ctxsave in %s\n", dir)
		fmt.Println("Context database created at .ctxsave/context.db")
		return nil
	},
}
