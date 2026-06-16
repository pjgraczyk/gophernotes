package kernel

import (
	"bufio"
	"io"
	"strings"

	"github.com/pjgraczyk/gomacro/base"
)

func (kernel *Kernel) checkComplete(code string) (status, indent string) {
	status, indent = "unknown", ""

	if len(code) == 0 {
		return status, indent
	}
	readline := base.MakeBufReadline(bufio.NewReader(strings.NewReader(code)))
	for {
		_, _, err := base.ReadMultiline(readline, base.ReadOptions(0), "")
		if err == io.EOF {
			return "complete", indent
		} else if err == io.ErrUnexpectedEOF {
			return "incomplete", indent
		} else if err != nil {
			return "invalid", indent
		}
	}
}
