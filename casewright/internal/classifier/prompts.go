package classifier

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// DefaultPromptVersion is the system prompt used when PROMPT_VERSION is unset.
// Promote a new version only after it beats this one on the eval set.
const DefaultPromptVersion = "v2"

//go:embed prompts/classifier_*.txt
var promptFS embed.FS

// Prompt returns the system prompt for a version such as "v1".
func Prompt(version string) (string, error) {
	b, err := promptFS.ReadFile("prompts/classifier_" + version + ".txt")
	if err != nil {
		return "", fmt.Errorf("unknown prompt version %q (have %s)", version, strings.Join(PromptVersions(), ", "))
	}
	return string(b), nil
}

// PromptVersions lists the embedded prompt versions.
func PromptVersions() []string {
	names, _ := fs.Glob(promptFS, "prompts/classifier_*.txt")
	versions := make([]string, 0, len(names))
	for _, n := range names {
		versions = append(versions, strings.TrimSuffix(strings.TrimPrefix(n, "prompts/classifier_"), ".txt"))
	}
	sort.Strings(versions)
	return versions
}
