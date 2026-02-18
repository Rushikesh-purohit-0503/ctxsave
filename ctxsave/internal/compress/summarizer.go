package compress

import (
	"fmt"
	"strings"

	"ctxsave/internal/store"
)

const (
	LevelRaw        = "raw"
	LevelDetailed   = "detailed"
	LevelCompressed = "compressed"
	LevelUltra      = "ultra"
)

type Summarizer struct{}

func NewSummarizer() *Summarizer {
	return &Summarizer{}
}

func (s *Summarizer) Summarize(entries []Entry) map[string]string {
	return map[string]string{
		LevelRaw:        s.buildRaw(entries),
		LevelDetailed:   s.buildDetailed(entries),
		LevelCompressed: s.buildCompressed(entries),
		LevelUltra:      s.buildUltra(entries),
	}
}

func (s *Summarizer) BestFit(summaries map[string]string, budget int, family ModelFamily) (string, string) {
	levels := []string{LevelRaw, LevelDetailed, LevelCompressed, LevelUltra}
	for _, lvl := range levels {
		text := summaries[lvl]
		tokens := EstimateTokens(text, family)
		if tokens <= budget {
			return lvl, text
		}
	}
	return LevelUltra, summaries[LevelUltra]
}

type Entry = store.Entry

func (s *Summarizer) buildRaw(entries []Entry) string {
	var sb strings.Builder
	for _, e := range entries {
		content := e.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%s] %s\n\n", e.Type, content))
	}
	return sb.String()
}

func (s *Summarizer) buildDetailed(entries []Entry) string {
	grouped := groupByType(entries)
	var sb strings.Builder

	if items, ok := grouped[store.EntryDecision]; ok {
		sb.WriteString("### Key Decisions & Findings\n")
		seen := make(map[string]bool)
		for _, e := range items {
			summary := extractMeaningfulLine(e.Content)
			if summary == "" || seen[summary] {
				continue
			}
			seen[summary] = true
			sb.WriteString(fmt.Sprintf("- %s\n", summary))
		}
		sb.WriteString("\n")
	}

	if items, ok := grouped[store.EntryCodeChange]; ok {
		edits := filterEdits(items)
		reads := filterReads(items)
		searches := filterSearches(items)

		if len(edits) > 0 {
			sb.WriteString("### Files Modified\n")
			for _, e := range edits {
				sb.WriteString(fmt.Sprintf("- %s\n", e))
			}
			sb.WriteString("\n")
		}

		if len(searches) > 0 {
			sb.WriteString("### Key Patterns Searched\n")
			seen := make(map[string]bool)
			for _, e := range searches {
				if !seen[e] {
					seen[e] = true
					sb.WriteString(fmt.Sprintf("- %s\n", e))
				}
			}
			sb.WriteString("\n")
		}

		if len(reads) > 0 {
			sb.WriteString("### Files Investigated\n")
			seen := make(map[string]bool)
			for _, e := range reads {
				if !seen[e] {
					seen[e] = true
					sb.WriteString(fmt.Sprintf("- %s\n", e))
				}
			}
			sb.WriteString("\n")
		}
	}

	if items, ok := grouped[store.EntryGitCommit]; ok {
		sb.WriteString("### Git Commits\n")
		for _, e := range items {
			sb.WriteString(fmt.Sprintf("- %s\n", firstLine(e.Content)))
		}
		sb.WriteString("\n")
	}

	if items, ok := grouped[store.EntryConversation]; ok {
		userQuestions := filterByMeta(items, "user")
		if len(userQuestions) > 0 {
			sb.WriteString("### Questions Discussed\n")
			seen := make(map[string]bool)
			for _, e := range userQuestions {
				line := cleanQuestionLine(e.Content)
				if line == "" || len(line) < 15 || seen[line] {
					continue
				}
				seen[line] = true
				sb.WriteString(fmt.Sprintf("- %s\n", truncateLine(line, 150)))
			}
			sb.WriteString("\n")
		}
	}

	if items, ok := grouped[store.EntryNote]; ok {
		sb.WriteString("### Notes\n")
		for _, e := range items {
			sb.WriteString(fmt.Sprintf("- %s\n", e.Content))
		}
		sb.WriteString("\n")
	}

	if items, ok := grouped[store.EntryFile]; ok {
		sb.WriteString("### Files Captured\n")
		for _, e := range items {
			sb.WriteString(fmt.Sprintf("- %s\n", firstLine(e.Content)))
		}
		sb.WriteString("\n")
	}

	if items, ok := grouped[store.EntryError]; ok {
		sb.WriteString("### Errors Encountered\n")
		seen := make(map[string]bool)
		for _, e := range items {
			line := firstLine(e.Content)
			if seen[line] {
				continue
			}
			seen[line] = true
			sb.WriteString(fmt.Sprintf("- %s\n", truncateLine(line, 150)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (s *Summarizer) buildCompressed(entries []Entry) string {
	grouped := groupByType(entries)
	var sb strings.Builder

	if items, ok := grouped[store.EntryDecision]; ok {
		sb.WriteString("**Key Findings:** ")
		var ds []string
		seen := make(map[string]bool)
		for _, e := range items {
			line := extractMeaningfulLine(e.Content)
			if line == "" || seen[line] {
				continue
			}
			seen[line] = true
			ds = append(ds, truncateLine(line, 100))
			if len(ds) >= 5 {
				break
			}
		}
		sb.WriteString(strings.Join(ds, "; "))
		sb.WriteString("\n\n")
	}

	if items, ok := grouped[store.EntryCodeChange]; ok {
		edits := filterEdits(items)
		if len(edits) > 0 {
			sb.WriteString(fmt.Sprintf("**Code Changes:** %d edits — ", len(edits)))
			sb.WriteString(strings.Join(edits, ", "))
			sb.WriteString("\n\n")
		}
	}

	if items, ok := grouped[store.EntryGitCommit]; ok {
		sb.WriteString(fmt.Sprintf("**Git:** %d commits", len(items)))
		if len(items) > 0 {
			sb.WriteString(fmt.Sprintf(" — latest: %s", firstLine(items[0].Content)))
		}
		sb.WriteString("\n\n")
	}

	if items, ok := grouped[store.EntryNote]; ok {
		sb.WriteString("**Notes:** ")
		var ns []string
		for _, e := range items {
			ns = append(ns, e.Content)
		}
		sb.WriteString(strings.Join(ns, "; "))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func (s *Summarizer) buildUltra(entries []Entry) string {
	grouped := groupByType(entries)

	var parts []string
	if items, ok := grouped[store.EntryDecision]; ok {
		parts = append(parts, fmt.Sprintf("%d decisions", len(items)))
	}
	if items, ok := grouped[store.EntryCodeChange]; ok {
		edits := filterEdits(items)
		parts = append(parts, fmt.Sprintf("%d code edits", len(edits)))
	}
	if items, ok := grouped[store.EntryGitCommit]; ok {
		parts = append(parts, fmt.Sprintf("%d commits", len(items)))
	}
	if items, ok := grouped[store.EntryNote]; ok {
		parts = append(parts, fmt.Sprintf("%d notes", len(items)))
	}

	files := extractEditedFiles(entries)
	filePart := ""
	if len(files) > 0 {
		if len(files) > 5 {
			files = files[:5]
		}
		filePart = fmt.Sprintf(" Files touched: %s.", strings.Join(files, ", "))
	}

	return fmt.Sprintf("Context: %s.%s", strings.Join(parts, ", "), filePart)
}

func groupByType(entries []Entry) map[store.EntryType][]Entry {
	m := make(map[store.EntryType][]Entry)
	for _, e := range entries {
		m[e.Type] = append(m[e.Type], e)
	}
	return m
}

func firstLine(s string) string {
	idx := strings.IndexByte(s, '\n')
	if idx < 0 {
		return s
	}
	return s[:idx]
}

func truncateLine(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// extractMeaningfulLine finds the first non-trivial line from content that conveys actual information.
func extractMeaningfulLine(s string) string {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) < 20 {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "now ") || strings.HasPrefix(lower, "let me ") ||
			strings.HasPrefix(lower, "good") || strings.HasPrefix(lower, "here's") ||
			strings.HasPrefix(lower, "okay") {
			continue
		}
		return truncateLine(line, 200)
	}
	fl := firstLine(s)
	if len(fl) > 20 {
		return truncateLine(fl, 200)
	}
	return ""
}

func filterByMeta(entries []Entry, role string) []Entry {
	needle := fmt.Sprintf(`"role":"%s"`, role)
	var result []Entry
	for _, e := range entries {
		if strings.Contains(e.Metadata, needle) {
			result = append(result, e)
		}
	}
	return result
}

// filterEdits returns deduplicated edit descriptions (only writes/edits, not reads).
func filterEdits(entries []Entry) []string {
	seen := make(map[string]bool)
	var edits []string
	for _, e := range entries {
		c := e.Content
		if strings.HasPrefix(c, "Edited ") || strings.HasPrefix(c, "Wrote ") {
			if !seen[c] {
				seen[c] = true
				edits = append(edits, c)
			}
		}
	}
	return edits
}

func filterReads(entries []Entry) []string {
	seen := make(map[string]bool)
	var reads []string
	for _, e := range entries {
		c := e.Content
		if strings.HasPrefix(c, "Read ") {
			path := strings.TrimPrefix(c, "Read ")
			if !seen[path] {
				seen[path] = true
				reads = append(reads, path)
			}
		}
	}
	return reads
}

func filterSearches(entries []Entry) []string {
	seen := make(map[string]bool)
	var searches []string
	for _, e := range entries {
		c := e.Content
		if strings.HasPrefix(c, "Searched for ") {
			if !seen[c] {
				seen[c] = true
				searches = append(searches, c)
			}
		}
	}
	return searches
}

// cleanQuestionLine extracts a clean question from user content, skipping code blocks.
func cleanQuestionLine(s string) string {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || len(line) < 15 {
			continue
		}
		if looksLikeCode(line) {
			continue
		}
		if strings.HasPrefix(line, "@") {
			line = strings.TrimSpace(line[strings.Index(line, " ")+1:])
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

func looksLikeCode(s string) bool {
	trimmed := strings.TrimSpace(s)

	if strings.HasSuffix(trimmed, ";") || strings.HasSuffix(trimmed, "{") ||
		strings.HasSuffix(trimmed, "}") || strings.HasSuffix(trimmed, "},") ||
		strings.HasSuffix(trimmed, ");") || strings.HasSuffix(trimmed, "});") {
		return true
	}

	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "*/") {
		return true
	}

	codeIndicators := []string{
		"const ", "let ", "var ", "function ", "async ", "await ",
		"if (", "if(", "} else", "=>", "===", "!==",
		".find(", ".filter(", ".map(", ".push(", ".select(",
		"new ObjectId(", "new mongoose.", "req.", "res.",
		"L1:", "L2:", "L3:", "L4:", "L5:", "L6:", "L7:", "L8:", "L9:",
		"return (", "return {", "return [",
		"success:", "message:", "resType:",
		"sfsrtlp0141@",
		"**Project:**",
	}
	for _, ind := range codeIndicators {
		if strings.Contains(s, ind) {
			return true
		}
	}

	// Property assignment: "key: value," or "key: value"
	if strings.Contains(trimmed, ": ") && !strings.Contains(trimmed, "? ") {
		colonIdx := strings.Index(trimmed, ":")
		if colonIdx > 0 && colonIdx < 30 {
			before := trimmed[:colonIdx]
			if !strings.Contains(before, " ") || strings.HasPrefix(before, "[") {
				return true
			}
		}
	}

	// If more than 20% of characters are special code chars, it's likely code
	specials := 0
	for _, r := range trimmed {
		if r == '{' || r == '}' || r == '(' || r == ')' || r == '[' || r == ']' ||
			r == '=' || r == ';' || r == ':' || r == '.' || r == '>' || r == '<' {
			specials++
		}
	}
	if len(trimmed) > 0 && float64(specials)/float64(len(trimmed)) > 0.20 {
		return true
	}

	return false
}

// extractEditedFiles pulls unique shortened file paths from code change entries.
func extractEditedFiles(entries []Entry) []string {
	seen := make(map[string]bool)
	var files []string
	for _, e := range entries {
		if e.Type != store.EntryCodeChange {
			continue
		}
		c := e.Content
		var path string
		if strings.HasPrefix(c, "Edited ") {
			path = strings.TrimPrefix(c, "Edited ")
		} else if strings.HasPrefix(c, "Wrote ") {
			path = strings.TrimPrefix(c, "Wrote ")
		}
		if path != "" && !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}
	return files
}
