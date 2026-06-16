package kernel

import (
	interp "github.com/pjgraczyk/gomacro/fast"

	"github.com/gopherdata/gophernotes/internal/messaging"
)

type Completion struct {
	class, name, typ string
}

type CompletionResponse struct {
	partial     int
	completions []Completion
}

func handleCompleteRequest(ir *interp.Interp, receipt messaging.MsgReceipt) error {
	reqcontent := receipt.Msg.Content.(map[string]interface{})
	code := reqcontent["code"].(string)
	cursorPos := int(reqcontent["cursor_pos"].(float64))

	prefix, matches, _ := ir.CompleteWords(code, cursorPos)

	content := make(map[string]interface{})

	if len(matches) == 0 {
		content["ename"] = "ERROR"
		content["evalue"] = "no completions found"
		content["traceback"] = nil
		content["status"] = "error"
	} else {
		partialWord := interp.TailIdentifier(prefix)
		content["cursor_start"] = float64(len(prefix) - len(partialWord))
		content["cursor_end"] = float64(cursorPos)
		content["matches"] = matches
		content["status"] = "ok"
	}

	return receipt.Reply("complete_reply", content)
}
