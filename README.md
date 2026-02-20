# ctxsave — Cross-Model Context Saver

A Go CLI tool that captures context from Cursor sessions, git history, and manual notes, stores it in a local SQLite database, and generates compressed context prompts you can paste into a new session with any AI model.

## Problem

Limited context windows on free plans mean you lose project knowledge when switching between models or starting new sessions. There's no way to carry forward what was discussed, what code was changed, or what decisions were made.

## How It Works

1. **Capture** context from Cursor transcripts, git diffs/logs, and manual notes
2. **Store** it in a local SQLite DB (`.ctxsave/context.db`) per project
3. **Generate** a compressed context prompt tailored to the target model's context window
4. **Paste** the prompt as the first message in a new chat with any model

## Supported Models

| Key      | Model              | Context  | Description                              |
|----------|--------------------|----------|------------------------------------------|
| `gemini` | Gemini 2.5 Flash   | 1M       | Free, massive context window             |
| `opus`   | Claude Opus 4.6    | 200K     | Deep reasoning, best for complex tasks   |
| `sonnet` | Claude Sonnet 4    | 200K     | Best coding model — fast, high quality   |
| `gpt4o`  | GPT-4o             | 128K     | Strong general-purpose coding model      |

## Installation

```bash
# Build from source
cd ctxsave
go build -o ctxsave .

# Install globally
sudo cp ctxsave /usr/local/bin/

# Or install via go
go install .
```

Requires Go 1.21+.

## Quick Start

```bash
# Initialize in your project
cd your-project
ctxsave init

# Capture context
ctxsave capture note "decided to use JWT for authentication"
ctxsave capture git --since 4h
ctxsave capture cursor ~/.cursor/projects/*/agent-transcripts/*.jsonl

# Generate a context prompt and copy to clipboard
ctxsave generate --model sonnet --copy

# Paste into a new AI session — the model picks up where you left off
```

## Commands

### `ctxsave init`
Initialize context tracking in the current directory. Creates `.ctxsave/` with a SQLite database.

### `ctxsave capture cursor <path>` or `ctxsave capture cursor (auto detect the JSONL file in default dircttory)`
Parse a Cursor agent transcript JSONL file. Extracts conversations, decisions, tool calls (file edits, commands), and errors.

```bash
ctxsave capture cursor ~/.cursor/projects/my-project/agent-transcripts/abc123.jsonl
```
```bash
ctxsave capture cursor
```

### `ctxsave capture git`
Capture recent git history (commits and diffs).

```bash
ctxsave capture git --since 4h
ctxsave capture git --commits 20
```

### `ctxsave capture note "text"`
Add a manual context note.

```bash
ctxsave capture note "refactored auth to use middleware pattern"
```

### `ctxsave capture file <path>`
Capture a file's content as context, with an optional tag.

```bash
ctxsave capture file architecture.md --tag architecture
```

### `ctxsave sessions`
List all captured context sessions with timestamps, sources, and entry counts.

### `ctxsave show <session-id>`
Show full details of a specific session including all entries.

### `ctxsave generate`
Generate a context prompt for pasting into a new AI session.

```bash
ctxsave generate --model gemini --budget 16000 --copy
ctxsave generate --model sonnet --out context.md
ctxsave generate --model opus
```

Flags:
- `--model` — target model key (default: `sonnet`)
- `--budget` — token budget (default: auto based on model)
- `--copy` — copy to clipboard
- `--out` — write to file

### `ctxsave models`
List supported models with their context window sizes.

## Context Compression

The summarizer has 4 levels that progressively condense your context:

| Level        | What's included                                     |
|-------------|-----------------------------------------------------|
| `raw`       | Everything — full content of all entries            |
| `detailed`  | Grouped by type, first-line summaries, key content  |
| `compressed`| Key points only, counts, latest items               |
| `ultra`     | One-line briefing with counts and key files         |

When generating, ctxsave automatically picks the richest level that fits within your token budget.

## Project Structure

```
ctxsave/
├── main.go
├── cmd/
│   ├── root.go          # Cobra root command
│   ├── init.go          # ctxsave init
│   ├── capture.go       # ctxsave capture {cursor|git|note|file}
│   ├── sessions.go      # ctxsave sessions / show
│   ├── generate.go      # ctxsave generate --model X
│   └── models.go        # ctxsave models
├── internal/
│   ├── capture/
│   │   ├── cursor.go    # Cursor transcript JSONL parser
│   │   ├── git.go       # Git log + diff capture
│   │   └── manual.go    # Manual note and file capture
│   ├── store/
│   │   ├── sqlite.go    # SQLite operations
│   │   └── models.go    # Session, Entry, Summary types
│   ├── compress/
│   │   ├── summarizer.go # Extractive summarization
│   │   └── tokens.go    # Per-model-family token estimation
│   └── generate/
│       ├── prompt.go    # Prompt builder
│       └── profiles.go  # Model profiles
├── go.mod
└── README.md
```

## License

MIT
