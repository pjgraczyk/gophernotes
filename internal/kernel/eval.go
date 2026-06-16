package kernel

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/pjgraczyk/gomacro/ast2"
	"github.com/pjgraczyk/gomacro/base"
	basereflect "github.com/pjgraczyk/gomacro/base/reflect"
	interp "github.com/pjgraczyk/gomacro/fast"
	"github.com/pjgraczyk/gomacro/xreflect"

	"github.com/gopherdata/gophernotes/internal/messaging"
	"github.com/gopherdata/gophernotes/internal/rendering"
)

func (kernel *Kernel) handleExecuteRequest(receipt messaging.MsgReceipt) error {
	reqcontent := receipt.Msg.Content.(map[string]interface{})
	code := reqcontent["code"].(string)
	silent := reqcontent["silent"].(bool)

	if !silent {
		ExecCounter++
	}

	content := make(map[string]interface{})
	content["execution_count"] = ExecCounter

	if err := receipt.PublishExecutionInput(ExecCounter, code); err != nil {
		log.Printf("Error publishing execution input: %v\n", err)
	}

	oldStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		return err
	}
	os.Stdout = wOut

	oldStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	if err != nil {
		return err
	}
	os.Stderr = wErr

	var writersWG sync.WaitGroup
	writersWG.Add(2)

	jupyterStdOut := messaging.JupyterStreamWriter{Stream: messaging.StreamStdout, Receipt: &receipt}
	jupyterStdErr := messaging.JupyterStreamWriter{Stream: messaging.StreamStderr, Receipt: &receipt}
	outerr := messaging.OutErr{Out: &jupyterStdOut, Err: &jupyterStdErr}

	go func() {
		defer writersWG.Done()
		io.Copy(&jupyterStdOut, rOut)
	}()

	go func() {
		defer writersWG.Done()
		io.Copy(&jupyterStdErr, rErr)
	}()

	ir := kernel.ir
	displayPlace := ir.ValueOf("Display")
	displayPlace.Set(xreflect.ValueOf(receipt.PublishDisplayData))
	defer func() {
		displayPlace.Set(xreflect.ValueOf(rendering.StubDisplay))
	}()

	vals, types, executionErr := doEval(ir, outerr, code)

	wOut.Close()
	os.Stdout = oldStdout

	wErr.Close()
	os.Stderr = oldStderr

	writersWG.Wait()

	if executionErr == nil {
		data := kernel.autoRenderResults(vals, types)

		content["status"] = "ok"
		content["user_expressions"] = make(map[string]string)

		if !silent && len(data.Data) != 0 {
			if err := receipt.PublishExecutionResult(ExecCounter, data); err != nil {
				log.Printf("Error publishing execution result: %v\n", err)
			}
		}
	} else {
		content["status"] = "error"
		content["ename"] = "ERROR"
		content["evalue"] = executionErr.Error()
		content["traceback"] = nil

		if err := receipt.PublishExecutionError(executionErr.Error(), []string{executionErr.Error()}); err != nil {
			log.Printf("Error publishing execution error: %v\n", err)
		}
	}

	return receipt.Reply("execute_reply", content)
}

func doEval(ir *interp.Interp, outerr messaging.OutErr, code string) (val []interface{}, typ []xreflect.Type, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	code = evalSpecialCommands(ir, outerr, code)

	compiler := ir.Comp

	compiler.Options &^= base.OptShowPrompt
	compiler.Options &^= base.OptTrapPanic
	compiler.Line = 0

	nodes := compiler.ParseBytes([]byte(code))
	srcAst := ast2.AnyToAst(nodes, "doEval")

	if srcAst == nil {
		return nil, nil, nil
	}

	compiledSrc := ir.CompileAst(srcAst)

	results, types := ir.RunExpr(compiledSrc)

	values := make([]interface{}, len(results))
	for i, result := range results {
		values[i] = basereflect.ValueInterface(result)
	}

	return values, types, nil
}
