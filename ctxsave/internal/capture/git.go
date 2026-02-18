package capture

import (
	"fmt"
	"os/exec"
	"strings"

	"ctxsave/internal/store"
)

func CaptureFromGit(st *store.Store, projectDir, project string, since string, maxCommits int) (*store.Session, error) {
	args := []string{"log", "--oneline", "--no-decorate"}
	if since != "" {
		args = append(args, "--since", since)
	}
	if maxCommits > 0 {
		args = append(args, fmt.Sprintf("-n%d", maxCommits))
	}
	args = append(args, "--format=%H|||%s|||%an|||%ai")

	cmd := exec.Command("git", args...)
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	logOutput := strings.TrimSpace(string(out))
	if logOutput == "" {
		return nil, fmt.Errorf("no git commits found matching criteria")
	}

	label := "git"
	if since != "" {
		label = fmt.Sprintf("git (since %s)", since)
	}
	sess, err := st.CreateSession("git", project, label)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(logOutput, "\n")
	for i, line := range lines {
		parts := strings.SplitN(line, "|||", 4)
		if len(parts) < 4 {
			continue
		}
		hash, subject, author, date := parts[0], parts[1], parts[2], parts[3]

		content := fmt.Sprintf("[%s] %s (by %s, %s)", hash[:8], subject, author, date)
		meta := fmt.Sprintf(`{"hash":"%s","author":"%s","date":"%s"}`, hash, author, date)

		if _, err := st.AddEntry(sess.ID, store.EntryGitCommit, content, meta, i); err != nil {
			return nil, err
		}
	}

	diffCmd := exec.Command("git", "diff", "--stat")
	diffCmd.Dir = projectDir
	diffOut, err := diffCmd.Output()
	if err == nil && len(strings.TrimSpace(string(diffOut))) > 0 {
		if _, err := st.AddEntry(sess.ID, store.EntryGitDiff, truncate(string(diffOut), 3000), `{"type":"unstaged"}`, len(lines)); err != nil {
			return nil, err
		}
	}

	stagedCmd := exec.Command("git", "diff", "--staged", "--stat")
	stagedCmd.Dir = projectDir
	stagedOut, err := stagedCmd.Output()
	if err == nil && len(strings.TrimSpace(string(stagedOut))) > 0 {
		if _, err := st.AddEntry(sess.ID, store.EntryGitDiff, truncate(string(stagedOut), 3000), `{"type":"staged"}`, len(lines)+1); err != nil {
			return nil, err
		}
	}

	return sess, nil
}
