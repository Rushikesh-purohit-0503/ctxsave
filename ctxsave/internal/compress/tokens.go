package compress

type ModelFamily string

const (
	FamilyClaude ModelFamily = "claude"
	FamilyGemini ModelFamily = "gemini"
	FamilyGPT    ModelFamily = "gpt"
)

// EstimateTokens uses character-based estimation which is more accurate for
// mixed code/text content than word-based. Most tokenizers average ~4 chars/token
// for English text, and ~3 chars/token for code-heavy content.
func EstimateTokens(text string, family ModelFamily) int {
	chars := len(text)
	charsPerToken := charsPerTokenRatio(family)
	return int(float64(chars) / charsPerToken)
}

func charsPerTokenRatio(family ModelFamily) float64 {
	switch family {
	case FamilyClaude:
		return 3.5
	case FamilyGemini:
		return 3.8
	case FamilyGPT:
		return 3.5
	default:
		return 3.5
	}
}
