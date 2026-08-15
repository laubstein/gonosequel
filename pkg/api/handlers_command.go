package api

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gofiber/fiber/v3"
)

// maxCommandLines caps how many commands a single /command request may
// run — the same safety-net spirit as pkg/client's maxAggregateResults: a
// pasted script shouldn't be able to tie up a request indefinitely.
const maxCommandLines = 200

type commandRequest struct {
	Script string `json:"script"`
}

type commandResult struct {
	Command string `json:"command"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleRunCommand executes a redis-cli-style script: one raw backend
// command per line, run in sequence against the database named by :db (not
// scoped to a collection — a raw command isn't limited to one key prefix).
// A failing line does not stop the ones after it, matching redis-cli's own
// non-interactive/pipe-mode behavior, so the response reports a result or
// error per line rather than failing the whole request on the first bad
// one.
//
// This is a POST, and --readonly's middleware rejects every non-GET
// request — deliberately: a raw command can be anything, including
// FLUSHDB, and there is no reliable way to tell a read command from a
// write one without a per-command allowlist, so --readonly blocks the
// whole endpoint rather than trying to allow "safe" commands through, the
// same reasoning already applied to /aggregate.
func (d *deps) handleRunCommand(c fiber.Ctx) error {
	db := c.Params("db")

	var req commandRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var lines []string
	for _, line := range strings.Split(req.Script, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > maxCommandLines {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("too many commands (max %d)", maxCommandLines))
	}

	client := currentClient(c)
	results := make([]commandResult, 0, len(lines))
	for _, line := range lines {
		args, err := tokenizeCommand(line)
		if err != nil {
			results = append(results, commandResult{Command: line, Error: err.Error()})
			continue
		}
		if len(args) == 0 {
			continue
		}

		result, err := client.RunCommand(c.Context(), db, args)
		if err != nil {
			results = append(results, commandResult{Command: line, Error: err.Error()})
			continue
		}
		results = append(results, commandResult{Command: line, Result: result})
	}

	return c.JSON(results)
}

// tokenizeCommand splits a single command line into arguments the way
// redis-cli does: whitespace-separated, with single- or double-quoted
// segments kept together (so `SET key "hello world"` is 3 args, not 4),
// and a backslash inside double quotes escaping the next character.
func tokenizeCommand(line string) ([]string, error) {
	var (
		args               []string
		cur                strings.Builder
		inSingle, inDouble bool
		hasCur             bool
	)

	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case inSingle:
			if r == '\'' {
				inSingle = false
			} else {
				cur.WriteRune(r)
			}
		case inDouble:
			if r == '\\' && i+1 < len(runes) {
				i++
				cur.WriteRune(runes[i])
			} else if r == '"' {
				inDouble = false
			} else {
				cur.WriteRune(r)
			}
		case r == '\'':
			inSingle = true
			hasCur = true
		case r == '"':
			inDouble = true
			hasCur = true
		case unicode.IsSpace(r):
			if hasCur {
				args = append(args, cur.String())
				cur.Reset()
				hasCur = false
			}
		default:
			cur.WriteRune(r)
			hasCur = true
		}
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote")
	}
	if hasCur {
		args = append(args, cur.String())
	}
	return args, nil
}
