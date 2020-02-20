package formula

import (
	"fmt"
	"runtime"
	"strings"
)

const (
	PathPattern = "%s/formulas/%s"
	DirFormula = "/formulas"
	DefaultConfig = "config.json"
	ConfigPattern = "%s/%s"
	CommandEnv = "COMMAND"
	BinPattern     = "%s%s"
	BinPathPattern = "%s/bin"
	windows        = "windows"
	EnvPattern = "%s=%s"
	CachePattern = "%s/.%s.cache"
	DefaultCacheNewLabel = "Type new value?"
	DefaultCacheQtd = 5
)

// Config type that represents formula config
type Config struct {
	Command     string  `json:"command"`
	Description string  `json:"description"`
	Inputs      []Input `json:"inputs"`
}

// Input type that represents input config
type Input struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Default string   `json:"default"`
	Label   string   `json:"label"`
	Items   []string `json:"items"`
	Cache   Cache    `json:"cache"`
}

type Cache struct {
	Active   bool   `json:"active"`
	Qtd      int    `json:"qtd"`
	NewLabel string `json:"newLabel"`
}

// Definition type that represents a Formula
type Definition struct {
	Path    string
	Bin     string
	Config  string
	RepoUrl string
}

// FormulaPath builds the formula path from ritchie home
func (d *Definition) FormulaPath(home string) string {
	return fmt.Sprintf(PathPattern, home, d.Path)
}

// BinName builds the bin name from definition params
func (d *Definition) BinName() string {
	so := runtime.GOOS
	suffix := ""
	if so == windows {
		suffix = ".exe"
	}
	binSO := strings.ReplaceAll(d.Bin, "${so}", so)

	return fmt.Sprintf(BinPattern, binSO, suffix)
}

// BinPath builds the bin path from formula path
func (d *Definition) BinPath(formula string) string {
	return fmt.Sprintf(BinPathPattern, formula)
}

// BinFilePath builds the bin file path from binPath and binName
func (d *Definition) BinFilePath(binPath, binName string) string {
	return fmt.Sprintf("%s/%s", binPath, binName)
}

// BinUrl builds the bin url
func (d *Definition) BinUrl() string {
	return fmt.Sprintf("%s/bin/%s.zip", d.RepoUrl, d.BinName())
}

// ConfigName resolver de config name
func (d *Definition) ConfigName() string {
	if d.Config != "" {
		return d.Config
	}
	return DefaultConfig
}

// ConfigPath builds the config path from formula path and config name
func (d *Definition) ConfigPath(formula, configName string) string {
	return fmt.Sprintf(ConfigPattern, formula, configName)
}

// ConfigUrl builds the config url
func (d *Definition) ConfigUrl(configName string) string {
	return fmt.Sprintf("%s/%s", d.RepoUrl, configName)
}

//go:generate $GOPATH/bin/moq -out mock_formulamanager.go . Manager

// Manager is an interface that we can use to perform formula operations
type Manager interface {
	Run(def Definition) error
}
