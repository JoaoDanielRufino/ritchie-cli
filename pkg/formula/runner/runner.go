package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/mattn/go-isatty"

	"github.com/ZupIT/ritchie-cli/pkg/api"
	"github.com/ZupIT/ritchie-cli/pkg/formula"
	"github.com/ZupIT/ritchie-cli/pkg/rcontext"
	"github.com/ZupIT/ritchie-cli/pkg/stream"
)

const (
	dockerCmd = "docker"
	envFile   = ".env"
)

type RunManager struct {
	formula.PostRunner
	formula.InputRunner
	formula.PreRunner
	file stream.FileWriteExistAppender
	ctx  rcontext.Finder
}

func NewFormulaRunner(
	postRun formula.PostRunner,
	input formula.InputRunner,
	preRun formula.PreRunner,
	file stream.FileWriteExistAppender,
	ctx rcontext.Finder,
) formula.Runner {
	return RunManager{
		PostRunner:  postRun,
		InputRunner: input,
		PreRunner:   preRun,
		file:        file,
		ctx:         ctx,
	}
}

func (ru RunManager) Run(def formula.Definition, inputType api.TermInputType, docker bool) error {
	setup, err := ru.PreRun(def, docker)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if !docker || setup.ContainerId == "" {
		cmd, err = ru.runLocal(setup, inputType)
		if err != nil {
			return err
		}
	} else {
		cmd, err = ru.runDocker(setup, inputType)
		if err != nil {
			return err
		}
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	if err := ru.PostRun(setup, docker); err != nil {
		return err
	}

	return nil
}

func (ru RunManager) runDocker(setup formula.Setup, inputType api.TermInputType) (*exec.Cmd, error) {
	volume := fmt.Sprintf("%s:/app", setup.Pwd)
	var args []string
	if isatty.IsTerminal(os.Stdout.Fd()) {
		args = []string{"run", "--rm", "-it", "--env-file", envFile, "-v", volume, "--name", setup.ContainerId, setup.ContainerId}
	} else {
		args = []string{"run", "--rm", "--env-file", envFile, "-v", volume, "--name", setup.ContainerId, setup.ContainerId}
	}

	cmd := exec.Command(dockerCmd, args...) // Run command "docker run -env-file .env -v "$(pwd):/app" --name (randomId) (randomId)"
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := ru.Inputs(cmd, setup, inputType); err != nil {
		return nil, err
	}

	if err := ru.setEnvs(cmd, "/app", true); err != nil {
		return nil, err
	}

	return cmd, nil
}

func (ru RunManager) runLocal(setup formula.Setup, inputType api.TermInputType) (*exec.Cmd, error) {
	formulaRun := path.Join(setup.TmpDir, setup.BinName)
	cmd := exec.Command(formulaRun)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = os.Environ()
	if err := ru.setEnvs(cmd, setup.Pwd, false); err != nil {
		return nil, err
	}

	if err := ru.Inputs(cmd, setup, inputType); err != nil {
		return nil, err
	}

	return cmd, nil
}

func (ru RunManager) setEnvs(cmd *exec.Cmd, pwd string, docker bool) error {
	ctx, err := ru.ctx.Find()
	if err != nil {
		return err
	}

	pwdEnv := fmt.Sprintf(formula.EnvPattern, formula.PwdEnv, pwd)
	ctxEnv := fmt.Sprintf(formula.EnvPattern, formula.CtxEnv, ctx.Current)
	cmd.Env = append(cmd.Env, pwdEnv, ctxEnv)

	if docker {
		for _, e := range cmd.Env { // Create a file named .env and add the environment variable inName=inValue
			if !ru.file.Exists(envFile) {
				if err := ru.file.Write(envFile, []byte(e+"\n")); err != nil {
					return err
				}
				continue
			}
			if err := ru.file.Append(envFile, []byte(e+"\n")); err != nil {
				return err
			}
		}
	}

	return nil
}
