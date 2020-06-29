package formula

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ZupIT/ritchie-cli/pkg/api"
	"github.com/ZupIT/ritchie-cli/pkg/file/fileutil"
	"github.com/ZupIT/ritchie-cli/pkg/formula/tpl/tpl_go"
	"github.com/ZupIT/ritchie-cli/pkg/formula/tpl/tpl_shell"
	"github.com/ZupIT/ritchie-cli/pkg/prompt"
)

const (
	localTreeFile     = "%s/tree/tree.json"
	nameModule        = "{{nameModule}}"
	nameBin           = "{{bin-name}}"
	nameBinFirstUpper = "{{bin-name-first-upper}}"
)

var (
	ErrDontStartWithRit = prompt.NewError("Rit formula's command needs to start with \"rit\" [ex.: rit group verb <noun>]")
	ErrTooShortCommand  = prompt.NewError("Rit formula's command needs at least 2 words following \"rit\" [ex.: rit group verb]")
	ErrRepeatedCommand  = prompt.NewError("this command already exists")
	ErrTreeJsonNotFound = prompt.NewError("tree.json not found")
	ErrMakefileNotFound = prompt.NewError("makefile not found")
)

type CreateManager struct {
	FormPath    string
	treeManager TreeManager
}

func NewCreator(homePath string, tm TreeManager) CreateManager {
	return CreateManager{FormPath: fmt.Sprintf(FormCreatePathPattern, homePath), treeManager: tm}
}

func (c CreateManager) Create(cf Create) (CreateManager, error) {
	_ = fileutil.CreateDirIfNotExists(c.FormPath, os.ModePerm)
	localRepoDir := cf.LocalRepoDir
	fCmd := cf.FormulaCmd
	lang := cf.Lang

	if localRepoDir != "" {

		if !existsTreeJson(localRepoDir) && existsMakefile(localRepoDir) {
			return CreateManager{}, ErrTreeJsonNotFound
		}
		if !existsMakefile(localRepoDir) && existsTreeJson(localRepoDir) {
			return CreateManager{}, ErrMakefileNotFound
		}

		c.FormPath = localRepoDir
	}

	trees, err := c.treeManager.Tree()
	if err != nil {
		return CreateManager{}, err
	}

	err = verifyCommand(fCmd, trees)
	if err != nil {
		return CreateManager{}, err
	}

	err = generateTreeJsonFile(c.FormPath, fCmd, lang)
	if err != nil {
		return CreateManager{}, err
	}

	if existsMakefile(c.FormPath) && existsTreeJson(c.FormPath) {
		err = generateFormulaFiles(c.FormPath, fCmd, lang, false)
		if err != nil {
			return CreateManager{}, err
		}
	} else {
		err = generateFormulaFiles(c.FormPath, fCmd, lang, true)
		if err != nil {
			return CreateManager{}, err
		}
	}

	return c, nil
}

func existsTreeJson(formPath string) bool {
	treePath := fmt.Sprintf(TreeCreatePathPattern, formPath)
	return fileutil.Exists(treePath)
}

func existsMakefile(formPath string) bool {
	makefilePath := fmt.Sprintf(MakefileCreatePathPattern, formPath, Makefile)
	return fileutil.Exists(makefilePath)
}

func generateFormulaFiles(formPath, fCmd, lang string, new bool) error {

	d := strings.Split(fCmd, " ")

	dirForm := strings.Join(d[1:], "/")
	formulaName := strings.Join(d[1:], "_")
	pkgName := d[len(d)-1]

	var dir string
	if new {
		dir = fmt.Sprintf("%s/%s", formPath, dirForm)
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			return err
		}
		err = createMakefileMain(formPath, dirForm, formulaName)
		if err != nil {
			return err
		}

	} else {
		dir = fmt.Sprintf("%s/%s", formPath, dirForm)
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			return err
		}
		err = changeMakefileMain(formPath, fCmd, formulaName)
		if err != nil {
			return err
		}
	}
	err := createConfigFile(dir)
	if err != nil {
		return err
	}
	err = createSrcFiles(dir, pkgName, lang)
	if err != nil {
		return err
	}
	return nil
}

func generateTreeJsonFile(formPath, fCmd, lang string) error {
	tree := Tree{Commands: []api.Command{}}
	dir := fmt.Sprintf(localTreeFile, formPath)
	jsonFile, err := fileutil.ReadFile(dir)
	if err != nil {
		if err := fileutil.CreateDirIfNotExists(filepath.Dir(dir), 0755); err != nil {
			return err
		}
	} else {
		if err := json.Unmarshal(jsonFile, &tree); err != nil {
			return err
		}
	}

	tree, err = updateTree(fCmd, tree, lang, 0)
	if err != nil {
		return err
	}
	treeJsonFile, _ := json.Marshal(&tree)
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, treeJsonFile, "", "\t"); err != nil {
		return err
	}

	return fileutil.WriteFile(dir, prettyJSON.Bytes())
}

func verifyCommand(fCmd string, trees map[string]Tree) error {
	s := strings.Split(fCmd, " ")

	if s[0] != "rit" {
		return ErrDontStartWithRit
	}

	if len(s) <= 2 {
		return ErrTooShortCommand
	}
	cp := fmt.Sprintf("root_%s", strings.Join(s[1:len(s)-1], "_"))
	u := s[len(s)-1]
	for _, v := range trees {
		for _, j := range v.Commands {
			if j.Parent == cp && j.Usage == u {
				return ErrRepeatedCommand

			}
		}
	}
	return nil
}

func changeMakefileMain(formPath, fCmd, fName string) error {
	d := strings.Split(fCmd, " ")
	dir := fmt.Sprintf("%s/%s", formPath, Makefile)
	tplFile, err := fileutil.ReadFile(dir)
	if err != nil {
		return err
	}
	variable := strings.ToUpper(fName) + "=" + strings.Join(d[1:], "/")
	tplFile = []byte(strings.ReplaceAll(string(tplFile), "\nFORMULAS=", "\n"+variable+"\nFORMULAS="))
	formulas := formulaValue(tplFile)
	tplFile = []byte(strings.ReplaceAll(string(tplFile), formulas, formulas+" $("+strings.ToUpper(fName)+")"))
	err = fileutil.WriteFile(dir, tplFile)
	if err != nil {
		return err
	}

	return nil
}

func formulaValue(file []byte) string {
	fileStr := string(file)
	return strings.Split(strings.Split(fileStr, "FORMULAS=")[1], "\n")[0]
}

func createMakefileMain(dir, dirForm, name string) error {
	tplFile := tpl_go.MakefileMain

	tplFile = strings.ReplaceAll(tplFile, "{{formName}}", strings.ToUpper(name))
	tplFile = strings.ReplaceAll(tplFile, "{{formPath}}", dirForm)

	err := createScripts(dir)
	if err != nil {
		return err
	}
	return fileutil.WriteFile(fmt.Sprintf("%s/Makefile", dir), []byte(tplFile))
}

func createScripts(dir string) error {
	tplFile := tpl_go.CopyBinConfig

	err := fileutil.WriteFilePerm(fmt.Sprintf("%s/copy-bin-configs.sh", dir), []byte(tplFile), 0755)
	if err != nil {
		return err
	}

	tplFile = tpl_go.UnzipBinConfigs

	return fileutil.WriteFilePerm(fmt.Sprintf("%s/unzip-bin-configs.sh", dir), []byte(tplFile), 0755)
}

func createSrcFiles(dir, pkg, lang string) error {
	srcDir := fmt.Sprintf("%s/src", dir)
	pkgDir := fmt.Sprintf("%s/%s", srcDir, pkg)
	err := fileutil.CreateDirIfNotExists(srcDir, os.ModePerm)
	if err != nil {
		return err
	}
	switch lang {
	case Golang:
		pkgDir := fmt.Sprintf("%s/pkg/%s", srcDir, pkg)
		golang := NewGo()
		if err := golang.Create(srcDir, pkg, pkgDir, dir); err != nil {
			return nil
		}
	case Javalang:
		java := NewJava()
		if err := java.Create(srcDir, pkg, pkgDir, dir); err != nil {
			return err
		}
	case Nodelang:
		node := NewNode()
		if err := node.Create(srcDir, pkg, pkgDir, dir); err != nil {
			return err
		}
	case Pythonlang:
		python := NewPython()
		if err := python.Create(srcDir, pkg, pkgDir, dir); err != nil {
			return err
		}
	default:
		shell := NewShell()
		if err = shell.Create(srcDir, pkg, pkgDir, dir); err != nil {
			return nil
		}
	}
	return nil
}

func createGenericFiles(srcDir, pkg, dir string, l Lang) error {
	err := createMainFile(srcDir, pkg, l.Main, l.FileFormat, l.StartFile, l.UpperCase)
	if err != nil {
		return err
	}
	err = createMakefileForm(srcDir, pkg, dir, l.Makefile, l.Compiled)
	if err != nil {
		return err
	}
	err = createDockerfile(pkg, srcDir, l.Dockerfile)
	if err != nil {
		return err
	}
	if err := createUmask(srcDir); err != nil {
		return err
	}

	return nil
}

func createPkgDir(pkgDir string) error {
	return fileutil.CreateDirIfNotExists(pkgDir, os.ModePerm)
}

func createRunTemplate(dir, tpl string) error {
	return fileutil.WriteFilePerm(fmt.Sprintf("%s/run_template", dir), []byte(tpl), 0777)
}

func createMakefileForm(dir, name, pathName, tpl string, compiled bool) error {
	if compiled {
		tpl = strings.ReplaceAll(tpl, "{{name}}", name)
		tpl = strings.ReplaceAll(tpl, "{{form-path}}", pathName)
		return fileutil.WriteFile(fmt.Sprintf("%s/Makefile", dir), []byte(tpl))
	}
	tpl = strings.ReplaceAll(tpl, nameBin, name)
	return fileutil.WriteFile(fmt.Sprintf("%s/Makefile", dir), []byte(tpl))
}

func createDockerfile(pkg, dir, tpl string) error {
	tpl = strings.ReplaceAll(tpl, "{{bin-name}}", pkg)
	return fileutil.WriteFile(fmt.Sprintf("%s/Dockerfile", dir), []byte(tpl))
}

func createUmask(dir string) error {
	uMaskFile := fmt.Sprintf("%s/set_umask.sh", dir)
	return fileutil.WriteFile(uMaskFile, []byte(tpl_shell.Umask))
}

func createGoModFile(dir, pkg string) error {
	tplFile := tpl_go.GoMod
	tplFile = strings.ReplaceAll(tplFile, nameModule, pkg)
	return fileutil.WriteFile(fmt.Sprintf("%s/go.mod", dir), []byte(tplFile))
}

func createMainFile(dir, pkg, tpl, fileFormat, startFile string, uc bool) error {
	if uc {
		tpl = strings.ReplaceAll(tpl, nameBin, pkg)
		tpl = strings.ReplaceAll(tpl, nameBinFirstUpper, strings.Title(strings.ToLower(pkg)))
		return fileutil.WriteFile(fmt.Sprintf("%s/%s.%s", dir, startFile, fileFormat), []byte(tpl))
	}
	tpl = strings.ReplaceAll(tpl, nameModule, pkg)
	tpl = strings.ReplaceAll(tpl, nameBin, pkg)
	return fileutil.WriteFilePerm(fmt.Sprintf("%s/%s.%s", dir, startFile, fileFormat), []byte(tpl), 0777)
}

func createConfigFile(dir string) error {
	tplFile := tpl_go.Config
	return fileutil.WriteFile(fmt.Sprintf("%s/config.json", dir), []byte(tplFile))
}

func updateTree(fCmd string, t Tree, lang string, i int) (Tree, error) {
	fc := splitFormulaCommand(fCmd)
	parent := generateParent(fc, i)

	command := funk.Filter(t.Commands, func(command api.Command) bool {
		return command.Usage == fc[i] && command.Parent == parent
	}).([]api.Command)

	if len(fc)-1 == i {
		if len(command) == 0 {
			pathValue := strings.Join(fc, "/")
			fn := fc[len(fc)-1]
			var commands []api.Command
			if lang == Pythonlang {
				commands = append(t.Commands, api.Command{
					Usage: fn,
					Help:  fmt.Sprintf("%s %s", fc[i-1], fc[i]),
					Formula: api.Formula{
						Path:   pathValue,
						Bin:    "main.py",
						LBin:   "main.py",
						MBin:   "main.py",
						WBin:   fmt.Sprintf("%s.bat", fn),
						Bundle: "${so}.zip",
						Config: "config.json",
					},
					Parent: parent,
				})
			} else if lang == Golang {
				commands = append(t.Commands, api.Command{
					Usage: fn,
					Help:  fmt.Sprintf("%s %s", fc[i-1], fc[i]),
					Formula: api.Formula{
						Path:   pathValue,
						Bin:    fmt.Sprintf("%s-${so}", fn),
						LBin:   fmt.Sprintf("%s-${so}", fn),
						MBin:   fmt.Sprintf("%s-${so}", fn),
						WBin:   fmt.Sprintf("%s-${so}.exe", fn),
						Bundle: "${so}.zip",
						Config: "config.json",
					},
					Parent: parent,
				})
			} else {
				commands = append(t.Commands, api.Command{
					Usage: fn,
					Help:  fmt.Sprintf("%s %s", fc[i-1], fc[i]),
					Formula: api.Formula{
						Path:   pathValue,
						Bin:    fmt.Sprintf("%s.sh", fn),
						LBin:   fmt.Sprintf("%s.sh", fn),
						MBin:   fmt.Sprintf("%s.sh", fn),
						WBin:   fmt.Sprintf("%s.bat", fn),
						Bundle: "${so}.zip",
						Config: "config.json",
					},
					Parent: parent,
				})
			}
			t.Commands = commands
			return t, nil
		} else {
			return Tree{}, prompt.NewError("Command already exist ")
		}

	} else {
		if len(command) == 0 {
			commands := append(t.Commands, api.Command{
				Usage:  fc[i],
				Help:   generateCommandHelp(parent, fc, i),
				Parent: parent,
			})
			t.Commands = commands
		}
	}

	return updateTree(fCmd, t, lang, i+1)
}

func generateCommandHelp(parent string, fc []string, i int) string {
	var help string
	if parent != "root" {
		help = fc[i-1] + " " + fc[i]
	} else {
		help = fc[i] + " commands"
	}
	return help
}

func splitFormulaCommand(formulaCommand string) []string {
	return funk.Filter(strings.Split(formulaCommand, " "), func(input string) bool {
		return input != "" && input != "rit"
	}).([]string)
}

func generateParent(fc []string, index int) string {
	if index > 0 {
		return "root_" + strings.Join(fc[0:index], "_")
	} else {
		return "root"
	}
}

func createPackageJson(dir, tpl string) error {
	return fileutil.WriteFile(fmt.Sprintf("%s/package.json", dir), []byte(tpl))
}
