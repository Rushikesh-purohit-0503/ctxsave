package capture

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ctxsave/internal/store"
)

type transcriptLine struct {
	Role    string `json:"role"`
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

type parsedEntry struct {
	Type    store.EntryType
	Content string
	Meta    string
}

var decisionKeywords = []string{
	"decided", "decision", "chose", "going with", "opted for",
	"design choice", "the fix is", "the solution is", "root cause",
	"the problem is", "the issue is", "what needs to change",
}

var xmlTagsRe = regexp.MustCompile(`</?(?:attached_files|code_selection|user_query|terminal_selection|system_reminder|open_and_recently_viewed_files)[^>]*>`)
var thinkingPrefixRe = regexp.MustCompile(`(?m)^\[Thinking\]\s*`)
var lineNumberRe = regexp.MustCompile(`(?m)^\s*(?:L?\d+\|)`)

func FindCursorTranscriptsDir(projectDir string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	cursorProjectsDir := filepath.Join(homeDir, ".cursor", "projects")
	if _, err := os.Stat(cursorProjectsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("cursor projects directory not found at %s", cursorProjectsDir)
	}

	folderName := strings.TrimPrefix(projectDir, "/")
	folderName = strings.ReplaceAll(folderName, "/", "-")

	transcriptsDir := filepath.Join(cursorProjectsDir, folderName, "agent-transcripts")
	if _, err := os.Stat(transcriptsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("no Cursor transcripts found for this project at %s", transcriptsDir)
	}

	return transcriptsDir, nil
}

func FindTranscriptFiles(transcriptsDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(transcriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".txt" || ext == ".jsonl" {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

type AutoCaptureResult struct {
	Captured int
	Skipped  int
	Errors   []string
}

func CaptureAllFromCursor(st *store.Store, projectDir, project string) (*AutoCaptureResult, error) {
	transcriptsDir, err := FindCursorTranscriptsDir(projectDir)
	if err != nil {
		return nil, err
	}

	files, err := FindTranscriptFiles(transcriptsDir)
	if err != nil {
		return nil, fmt.Errorf("scan transcripts dir: %w", err)
	}

	result := &AutoCaptureResult{}

	for _, file := range files {
		processed, err := st.IsTranscriptProcessed(file)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: check error: %v", filepath.Base(file), err))
			continue
		}
		if processed {
			result.Skipped++
			continue
		}

		sess, err := CaptureFromCursor(st, file, project)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filepath.Base(file), err))
			continue
		}

		info, _ := os.Stat(file)
		var size int64
		if info != nil {
			size = info.Size()
		}
		if err := st.MarkTranscriptProcessed(file, sess.ID, size); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: mark processed: %v", filepath.Base(file), err))
		}

		result.Captured++
	}

	return result, nil
}

func CaptureFromCursor(st *store.Store, transcriptPath, project string) (*store.Session, error) {
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}

	label := filepath.Base(transcriptPath)
	sess, err := st.CreateSession("cursor", project, label)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(transcriptPath)
	var parseErr error
	if ext == ".jsonl" {
		parseErr = parseJSONL(st, sess.ID, data)
	} else {
		parseErr = parseTextTranscript(st, sess.ID, data)
	}

	if parseErr != nil {
		return nil, parseErr
	}

	info, _ := os.Stat(transcriptPath)
	var size int64
	if info != nil {
		size = info.Size()
	}
	_ = st.MarkTranscriptProcessed(transcriptPath, sess.ID, size)

	return sess, nil
}

func parseJSONL(st *store.Store, sessionID string, data []byte) error {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	orderIdx := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var tl transcriptLine
		if err := json.Unmarshal(line, &tl); err != nil {
			continue
		}

		entries := extractEntries(tl)
		for _, pe := range entries {
			if _, err := st.AddEntry(sessionID, pe.Type, pe.Content, pe.Meta, orderIdx); err != nil {
				return err
			}
			orderIdx++
		}
	}
	return scanner.Err()
}

func parseTextTranscript(st *store.Store, sessionID string, data []byte) error {
	content := string(data)
	lines := strings.Split(content, "\n")

	type section struct {
		role    string
		content strings.Builder
	}

	var sections []section
	var current *section

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "user:" {
			sections = append(sections, section{role: "user"})
			current = &sections[len(sections)-1]
			continue
		}
		if trimmed == "assistant:" {
			sections = append(sections, section{role: "assistant"})
			current = &sections[len(sections)-1]
			continue
		}
		if strings.HasPrefix(trimmed, "[Tool call]") {
			sections = append(sections, section{role: "tool_call"})
			current = &sections[len(sections)-1]
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "[Tool call]"))
			if rest != "" {
				current.content.WriteString(rest)
				current.content.WriteByte('\n')
			}
			continue
		}
		if strings.HasPrefix(trimmed, "[Tool result]") {
			sections = append(sections, section{role: "tool_result"})
			current = &sections[len(sections)-1]
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "[Tool result]"))
			if rest != "" {
				current.content.WriteString(rest)
				current.content.WriteByte('\n')
			}
			continue
		}

		if current != nil {
			current.content.WriteString(line)
			current.content.WriteByte('\n')
		}
	}

	orderIdx := 0
	for _, sec := range sections {
		text := strings.TrimSpace(sec.content.String())
		if text == "" {
			continue
		}

		var entryType store.EntryType
		var meta string

		switch sec.role {
		case "user":
			text = cleanContent(text)
			text = extractUserQuery(text)
			if text == "" {
				continue
			}
			entryType = store.EntryConversation
			meta = `{"role":"user"}`
			text = truncate(text, 2000)

		case "assistant":
			text = cleanContent(text)
			if text == "" || isMetaNoise(text) {
				continue
			}
			entryType = classifyAssistantText(text)
			meta = `{"role":"assistant"}`
			text = truncate(text, 3000)

		case "tool_call":
			summary := summarizeToolCall(text)
			if summary == "" {
				continue
			}
			entryType = store.EntryCodeChange
			meta = `{"source":"tool_call"}`
			text = summary

		case "tool_result":
			lower := strings.ToLower(text)
			if strings.Contains(lower, "error") || strings.Contains(lower, "failed") {
				entryType = store.EntryError
				meta = `{"source":"tool_result"}`
				text = cleanContent(truncate(text, 500))
			} else {
				continue
			}

		default:
			continue
		}

		if _, err := st.AddEntry(sessionID, entryType, text, meta, orderIdx); err != nil {
			return err
		}
		orderIdx++
	}

	return nil
}

func extractEntries(tl transcriptLine) []parsedEntry {
	var entries []parsedEntry

	text := extractText(tl)
	if text == "" {
		return entries
	}

	switch tl.Role {
	case "user":
		cleaned := cleanContent(text)
		query := extractUserQuery(cleaned)
		if query == "" {
			return entries
		}
		entries = append(entries, parsedEntry{
			Type:    store.EntryConversation,
			Content: truncate(query, 2000),
			Meta:    `{"role":"user"}`,
		})

	case "assistant":
		cleaned := cleanContent(text)
		if cleaned == "" || isMetaNoise(cleaned) {
			return entries
		}
		entryType := classifyAssistantText(cleaned)
		entries = append(entries, parsedEntry{
			Type:    entryType,
			Content: truncate(cleaned, 3000),
			Meta:    `{"role":"assistant"}`,
		})

	case "tool":
		lower := strings.ToLower(text)
		if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "exception") {
			entries = append(entries, parsedEntry{
				Type:    store.EntryError,
				Content: truncate(cleanContent(text), 500),
				Meta:    `{"source":"tool_result"}`,
			})
		}
	}

	return entries
}

// cleanContent strips XML tags, [Thinking] prefixes, inline line numbers, and collapses whitespace.
func cleanContent(s string) string {
	s = xmlTagsRe.ReplaceAllString(s, "")
	s = thinkingPrefixRe.ReplaceAllString(s, "")
	s = lineNumberRe.ReplaceAllString(s, "")

	// Collapse runs of blank lines
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(s)
}

// extractUserQuery pulls the actual user question from within <user_query> tags if present,
// otherwise returns the cleaned full text.
func extractUserQuery(s string) string {
	start := strings.Index(s, "<user_query>")
	end := strings.Index(s, "</user_query>")
	if start >= 0 && end > start {
		query := s[start+len("<user_query>") : end]
		return strings.TrimSpace(query)
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "yes" || s == "no" || s == "ok" {
		return ""
	}
	return s
}

// isMetaNoise returns true for content that's about the tooling itself, not the project.
func isMetaNoise(s string) bool {
	lower := strings.ToLower(s)
	noisePatterns := []string{
		"let me help you find your past chats",
		"let me look at the content of these chats",
		"let me also check if you have other cursor",
		"let me look for chats stored globally",
		"let me read the full transcript to extract",
		"let me explore your workspace",
		"dependencies installed",
		"all source files written",
		"build succeeded",
		"let me tidy deps",
		"let me clean up the test",
		"the commands seem to be running but not producing",
		"the issue is the `cd` in my build command",
		"let me separate the steps",
		"let me rebuild",
		"here's a summary of all your past cursor chats",
		"i found it. here's what happened with your",
		"all done. here's what was rebuilt",
		"your lost project",
		"ctxsave",
		".ctxsave/",
		"ctxsave init",
		"ctxsave capture",
		"ctxsave generate",
		"auto-captured",
		"second run correctly skips",
		"let me verify the binary",
		"fresh init and",
		"now let me test",
		"it automatically found and captured",
		"everything works end-to-end",
		"i have the full plan and architecture",
		"let me also install the binary",
		"now rebuild and test",
		"now let me do a full end-to-end test",
		"scaffold go project",
		"now the capture packages",
		"now the compress and generate",
		"now the cli commands",
		"good, directory structure created",
		"the output is still too noisy",
		"much better! now it's selecting",
		"getting much better",
		"the structure is clean now",
		"now it's selecting",
		"entry count dropped",
		"noise filtered",
		"token est",
		"ctxsave-related",
		"meta-conversation about",
		"raw level was selected",
		"compression level",
		"token budget",
	}
	for _, p := range noisePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func classifyAssistantText(text string) store.EntryType {
	lower := strings.ToLower(text)

	// Skip thinking-only blocks that just narrate what the AI is about to do
	if strings.HasPrefix(lower, "now ") && len(text) < 200 {
		return store.EntryConversation
	}

	for _, kw := range decisionKeywords {
		if strings.Contains(lower, kw) {
			// Only count as decision if there's enough substance
			if len(text) > 100 {
				return store.EntryDecision
			}
		}
	}
	return store.EntryConversation
}

// summarizeToolCall extracts a meaningful one-line summary from a tool call section.
func summarizeToolCall(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 0 {
		return ""
	}

	toolName := ""
	filePath := ""
	pattern := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if toolName == "" && line != "" {
			toolName = line
		}
		if strings.HasPrefix(line, "path:") {
			filePath = strings.TrimSpace(strings.TrimPrefix(line, "path:"))
		}
		if strings.HasPrefix(line, "pattern:") {
			pattern = strings.TrimSpace(strings.TrimPrefix(line, "pattern:"))
		}
	}

	switch {
	case strings.Contains(toolName, "StrReplace") && filePath != "":
		return fmt.Sprintf("Edited %s", shortenPath(filePath))
	case strings.Contains(toolName, "Write") && filePath != "":
		return fmt.Sprintf("Wrote %s", shortenPath(filePath))
	case strings.Contains(toolName, "Read") && filePath != "":
		return fmt.Sprintf("Read %s", shortenPath(filePath))
	case strings.Contains(toolName, "Grep") && pattern != "":
		if filePath != "" {
			return fmt.Sprintf("Searched for '%s' in %s", pattern, shortenPath(filePath))
		}
		return fmt.Sprintf("Searched for '%s'", pattern)
	case strings.Contains(toolName, "Glob"):
		return fmt.Sprintf("Found files matching pattern")
	case strings.Contains(toolName, "ReadLints"):
		return "Checked for lint errors"
	case filePath != "":
		return fmt.Sprintf("%s on %s", toolName, shortenPath(filePath))
	default:
		return ""
	}
}

func shortenPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) <= 3 {
		return p
	}
	return ".../" + strings.Join(parts[len(parts)-3:], "/")
}

func extractText(tl transcriptLine) string {
	var parts []string
	for _, c := range tl.Message.Content {
		if c.Type == "text" && c.Text != "" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
