package generate

import "ctxsave/internal/compress"

type ModelProfile struct {
	Key          string
	Name         string
	Family       compress.ModelFamily
	ContextLimit int
	Description  string
}

var Models = map[string]ModelProfile{
	"gemini": {
		Key:          "gemini",
		Name:         "Gemini 2.5 Flash",
		Family:       compress.FamilyGemini,
		ContextLimit: 1000000,
		Description:  "Free, massive context window",
	},
	"opus": {
		Key:          "opus",
		Name:         "Claude Opus 4.6",
		Family:       compress.FamilyClaude,
		ContextLimit: 200000,
		Description:  "Deep reasoning, best for complex tasks",
	},
	"sonnet": {
		Key:          "sonnet",
		Name:         "Claude Sonnet 4",
		Family:       compress.FamilyClaude,
		ContextLimit: 200000,
		Description:  "Best coding model — fast, high quality",
	},
	"gpt4o": {
		Key:          "gpt4o",
		Name:         "GPT-4o",
		Family:       compress.FamilyGPT,
		ContextLimit: 128000,
		Description:  "Strong general-purpose coding model",
	},
}

func GetModel(key string) (ModelProfile, bool) {
	m, ok := Models[key]
	return m, ok
}

func ListModels() []ModelProfile {
	order := []string{"gemini", "opus", "sonnet", "gpt4o"}
	var result []ModelProfile
	for _, k := range order {
		result = append(result, Models[k])
	}
	return result
}
