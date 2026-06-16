package kernel

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/pjgraczyk/gomacro/base"
	interp "github.com/pjgraczyk/gomacro/fast"

	"github.com/gopherdata/gophernotes/internal/messaging"
)

func evalSpecialCommands(ir *interp.Interp, outerr messaging.OutErr, code string) string {
	lines := strings.Split(code, "\n")
	stop := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) != 0 {
			switch line[0] {
			case '%':
				evalSpecialCommand(ir, outerr, line)
				lines[i] = ""
			case '$':
				evalShellCommand(ir, outerr, line)
				lines[i] = ""
			default:
				stop = true
			}
		}
		if stop {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func evalSpecialCommand(ir *interp.Interp, outerr messaging.OutErr, line string) {
	const help string = `
available special commands (%):
%cd [path]
%go111module {on|off}
%help

execute shell commands ($): $command [args...]
example:
$ls -l
`

	args := strings.SplitN(line, " ", 2)
	cmd := args[0]
	arg := ""
	if len(args) > 1 {
		arg = args[1]
	}
	switch cmd {
	case "%cd":
		if arg == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				panic(fmt.Errorf("error getting user home directory: %v", err))
			}
			arg = home
		}
		err := os.Chdir(arg)
		if err != nil {
			panic(fmt.Errorf("error setting current directory to %q: %v", arg, err))
		}
	case "%go111module":
		if arg == "on" {
			ir.Comp.CompGlobals.Options |= base.OptModuleImport
		} else if arg == "off" {
			ir.Comp.CompGlobals.Options &^= base.OptModuleImport
		} else {
			panic(fmt.Errorf("special command %s: expecting a single argument 'on' or 'off', found: %q", cmd, arg))
		}
	case "%help":
		outerr.Out.Write([]byte(help))
	default:
		panic(fmt.Errorf("unknown special command: %q\n%s", line, help))
	}
}

func evalShellCommand(ir *interp.Interp, outerr messaging.OutErr, line string) {
	args := strings.Fields(line[1:])
	if len(args) <= 0 {
		return
	}

	var writersWG sync.WaitGroup
	writersWG.Add(2)

	cmd := exec.Command(args[0], args[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(fmt.Errorf("Command.StdoutPipe() failed: %v", err))
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(fmt.Errorf("Command.StderrPipe() failed: %v", err))
	}

	go func() {
		defer writersWG.Done()
		io.Copy(outerr.Out, stdout)
	}()

	go func() {
		defer writersWG.Done()
		io.Copy(outerr.Err, stderr)
	}()

	err = cmd.Start()
	if err != nil {
		panic(fmt.Errorf("error starting command '%s': %v", line[1:], err))
	}

	err = cmd.Wait()
	if err != nil {
		panic(fmt.Errorf("error waiting for command '%s': %v", line[1:], err))
	}

	writersWG.Wait()
}
