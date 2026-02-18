package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sessionsCmd)
	rootCmd.AddCommand(showCmd)
}

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List captured context sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, _, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()

		sessions, err := st.ListSessions(50)
		if err != nil {
			return err
		}

		if len(sessions) == 0 {
			fmt.Println("No sessions yet — run 'ctxsave capture' first")
			return nil
		}

		fmt.Printf("%-18s %-20s %-10s %s\n", "ID", "CREATED", "SOURCE", "LABEL")
		fmt.Println("─────────────────────────────────────────────────────────────────")
		for _, s := range sessions {
			count, _ := st.CountEntries(s.ID)
			fmt.Printf("%-18s %-20s %-10s %s (%d entries)\n",
				s.ID,
				s.CreatedAt.Local().Format("2006-01-02 15:04"),
				s.Source,
				s.Label,
				count,
			)
		}
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show details of a specific session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		st, _, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()

		sess, err := st.GetSession(args[0])
		if err != nil {
			return fmt.Errorf("session %q not found", args[0])
		}

		entries, err := st.GetEntries(sess.ID)
		if err != nil {
			return err
		}

		fmt.Printf("Session: %s\n", sess.ID)
		fmt.Printf("Created: %s\n", sess.CreatedAt.Local().Format("2006-01-02 15:04:05"))
		fmt.Printf("Source:  %s\n", sess.Source)
		fmt.Printf("Label:   %s\n", sess.Label)
		fmt.Printf("Entries: %d\n\n", len(entries))

		for _, e := range entries {
			fmt.Printf("[%s] %s\n", e.Type, truncateShow(e.Content, 200))
			if e.Metadata != "" {
				fmt.Printf("  meta: %s\n", e.Metadata)
			}
			fmt.Println()
		}
		return nil
	},
}

func truncateShow(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
