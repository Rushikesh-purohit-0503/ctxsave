package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ctxsave/internal/capture"
	"ctxsave/internal/store"

	"github.com/spf13/cobra"
)

var (
	gitSince   string
	gitCommits int
	fileTag    string
)

func init() {
	rootCmd.AddCommand(captureCmd)
	captureCmd.AddCommand(captureCursorCmd)
	captureCmd.AddCommand(captureGitCmd)
	captureCmd.AddCommand(captureNoteCmd)
	captureCmd.AddCommand(captureFileCmd)

	captureGitCmd.Flags().StringVar(&gitSince, "since", "", "git log --since value (e.g. '4h', '1d')")
	captureGitCmd.Flags().IntVar(&gitCommits, "commits", 10, "max number of commits to capture")
	captureFileCmd.Flags().StringVar(&fileTag, "tag", "", "optional tag for the file")
}

var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Capture context from various sources",
}

var captureCursorCmd = &cobra.Command{
	Use:   "cursor [path-to-transcript]",
	Short: "Parse Cursor agent transcripts (auto-detects if no path given)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()

		if len(args) == 1 {
			sess, err := capture.CaptureFromCursor(st, args[0], project)
			if err != nil {
				return err
			}
			count, _ := st.CountEntries(sess.ID)
			fmt.Printf("Captured %d entries from Cursor transcript → session %s\n", count, sess.ID)
			return nil
		}

		dir, _ := os.Getwd()
		result, err := capture.CaptureAllFromCursor(st, dir, project)
		if err != nil {
			return err
		}

		fmt.Printf("Auto-captured %d new transcripts (%d already processed)\n", result.Captured, result.Skipped)
		for _, e := range result.Errors {
			fmt.Printf("  warning: %s\n", e)
		}
		if result.Captured == 0 && result.Skipped == 0 {
			fmt.Println("No Cursor transcripts found for this project.")
		}
		return nil
	},
}

var captureGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Capture recent git history",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()

		dir, _ := os.Getwd()
		sess, err := capture.CaptureFromGit(st, dir, project, gitSince, gitCommits)
		if err != nil {
			return err
		}

		count, _ := st.CountEntries(sess.ID)
		fmt.Printf("Captured %d entries from git → session %s\n", count, sess.ID)
		return nil
	},
}

var captureNoteCmd = &cobra.Command{
	Use:   "note \"your note text\"",
	Short: "Add a manual context note",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()

		note := strings.Join(args, " ")
		sess, err := capture.CaptureNote(st, project, note)
		if err != nil {
			return err
		}

		fmt.Printf("Note saved → session %s\n", sess.ID)
		return nil
	},
}

var captureFileCmd = &cobra.Command{
	Use:   "file <path>",
	Short: "Capture a file's content as context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()

		sess, err := capture.CaptureFile(st, project, args[0], fileTag)
		if err != nil {
			return err
		}

		fmt.Printf("File captured → session %s\n", sess.ID)
		return nil
	},
}

func openStore() (*store.Store, string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}

	ctxDir := filepath.Join(dir, ".ctxsave")
	if _, err := os.Stat(ctxDir); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("not initialized — run 'ctxsave init' first")
	}

	st, err := store.New(dir)
	if err != nil {
		return nil, "", err
	}

	project := filepath.Base(dir)
	return st, project, nil
}
