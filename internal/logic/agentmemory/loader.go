// 从磁盘的 memory 目录中加载项目级记忆和按路径匹配的规则记忆

package agentmemory

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	DefaultRoot       = "./memory"
	projectMemoryFile = "project.md" // 全局记忆，每次都加载
	rulesDir          = "rules"      // 规则文件，只有路径匹配的时候才加载
)

// Bundle 一次加载的最终结果
type Bundle struct {
	Project string
	Rules   []Rule
}

// Rule 一条规则文件被解析之后的结构
type Rule struct {
	Name    string
	Paths   []string
	Content string // 规则正文
}

type Loader struct {
	root string
}

func NewLoader(root string) *Loader {
	root = strings.TrimSpace(root)
	if root == "" {
		root = DefaultRoot
	}
	return &Loader{root: root}
}

// Load 总入口函数，从 memory 根目录中加载项目记忆和匹配规则，最后打包为一个 Bundle 返回
func (l *Loader) Load(paths []string) (Bundle, error) {
	project, err := l.loadProject()
	if err != nil {
		return Bundle{}, err
	}

	rules, err := l.loadRules(paths)
	if err != nil {
		return Bundle{}, err
	}

	return Bundle{
		Project: project,
		Rules:   rules,
	}, nil
}

// loadProject 加载全局记忆文件
func (l *Loader) loadProject() (string, error) {
	content, err := os.ReadFile(filepath.Join(l.root, projectMemoryFile))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// loadRules 加载规则文件，按路径筛选出本轮应生效的规则
func (l *Loader) loadRules(paths []string) ([]Rule, error) {
	rulesRoot := filepath.Join(l.root, rulesDir)
	entries := make([]string, 0, 8)

	err := filepath.WalkDir(rulesRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		entries = append(entries, path)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	sort.Strings(entries)

	matched := make([]Rule, 0, len(entries))
	for _, entry := range entries {
		rule, err := loadRuleFile(entry)
		if err != nil {
			return nil, err
		}
		if !rule.matches(paths) {
			continue
		}
		matched = append(matched, rule)
	}

	return matched, nil
}

// loadRuleFile 把一个规则文件解析为 Rule 结构
func loadRuleFile(filePath string) (Rule, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return Rule{}, err
	}

	paths, content := parseRuleFile(string(raw))
	return Rule{
		Name:    strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		Paths:   paths,
		Content: strings.TrimSpace(content),
	}, nil
}

// parseRuleFile 解析规则文件里面的 front matter
func parseRuleFile(raw string) ([]string, string) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return nil, raw
	}

	end := strings.Index(raw[4:], "\n---\n")
	if end < 0 {
		return nil, raw
	}

	frontMatter := raw[4 : 4+end]
	content := raw[4+end+5:]
	return parseRulePaths(frontMatter), content
}

func parseRulePaths(frontMatter string) []string {
	lines := strings.Split(frontMatter, "\n")
	paths := make([]string, 0, 4)
	inPaths := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "paths:":
			inPaths = true
		case inPaths && strings.HasPrefix(trimmed, "- "):
			pattern := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if pattern != "" {
				paths = append(paths, normalizePattern(pattern))
			}
		case trimmed == "":
		default:
			inPaths = false
		}
	}

	return paths
}

func (r Rule) matches(paths []string) bool {
	if len(r.Paths) == 0 {
		return true
	}
	if len(paths) == 0 {
		return false
	}

	for _, candidate := range paths {
		normalized := normalizePattern(candidate)
		for _, pattern := range r.Paths {
			if matchPattern(pattern, normalized) {
				return true
			}
		}
	}
	return false
}

func normalizePattern(v string) string {
	return strings.Trim(strings.ReplaceAll(v, "\\", "/"), "/")
}

func matchPattern(pattern, value string) bool {
	pattern = normalizePattern(pattern)
	value = normalizePattern(value)

	re := regexp.QuoteMeta(pattern)
	re = strings.ReplaceAll(re, "\\*\\*", ".*")
	re = strings.ReplaceAll(re, "\\*", "[^/]*")
	re = strings.ReplaceAll(re, "\\?", "[^/]")

	ok, err := regexp.MatchString("^"+re+"$", value)
	if err != nil {
		return false
	}
	return ok
}
