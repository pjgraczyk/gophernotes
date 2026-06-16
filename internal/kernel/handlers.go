package kernel

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/gopherdata/gophernotes/internal/messaging"
)

type kernelLanguageInfo struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	MIMEType          string `json:"mimetype"`
	FileExtension     string `json:"file_extension"`
	PygmentsLexer     string `json:"pygments_lexer"`
	CodeMirrorMode    string `json:"codemirror_mode"`
	NBConvertExporter string `json:"nbconvert_exporter"`
}

type helpLink struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type kernelInfo struct {
	ProtocolVersion       string             `json:"protocol_version"`
	Implementation        string             `json:"implementation"`
	ImplementationVersion string             `json:"implementation_version"`
	LanguageInfo          kernelLanguageInfo `json:"language_info"`
	Banner                string             `json:"banner"`
	HelpLinks             []helpLink         `json:"help_links"`
}

type shutdownReply struct {
	Restart bool `json:"restart"`
}

type isCompleteReply struct {
	Status string `json:"status"`
	Indent string `json:"indent"`
}

func (kernel *Kernel) handleShellMsg(receipt messaging.MsgReceipt) {
	if err := receipt.PublishKernelStatus(statusBusy); err != nil {
		log.Printf("Error publishing kernel status 'busy': %v\n", err)
	}
	defer func() {
		if err := receipt.PublishKernelStatus(statusIdle); err != nil {
			log.Printf("Error publishing kernel status 'idle': %v\n", err)
		}
	}()

	switch receipt.Msg.Header.MsgType {
	case "kernel_info_request":
		if err := sendKernelInfo(receipt); err != nil {
			log.Fatal(err)
		}
	case "is_complete_request":
		if err := kernel.handleIsCompleteRequest(receipt); err != nil {
			log.Fatal(err)
		}
	case "complete_request":
		if err := handleCompleteRequest(kernel.ir, receipt); err != nil {
			log.Fatal(err)
		}
	case "execute_request":
		if err := kernel.handleExecuteRequest(receipt); err != nil {
			log.Fatal(err)
		}
	case "shutdown_request":
		handleShutdownRequest(receipt)
	default:
		log.Println("Unhandled shell message: ", receipt.Msg.Header.MsgType)
	}
}

func sendKernelInfo(receipt messaging.MsgReceipt) error {
	return receipt.Reply("kernel_info_reply",
		kernelInfo{
			ProtocolVersion:       messaging.ProtocolVersion,
			Implementation:        "gophernotes",
			ImplementationVersion: Version,
			Banner:                fmt.Sprintf("Go kernel: gophernotes - v%s", Version),
			LanguageInfo: kernelLanguageInfo{
				Name:          "go",
				Version:       runtime.Version(),
				FileExtension: ".go",
			},
			HelpLinks: []helpLink{
				{Text: "Go", URL: "https://golang.org/"},
				{Text: "gophernotes", URL: "https://github.com/gopherdata/gophernotes"},
			},
		},
	)
}

func (kernel *Kernel) handleIsCompleteRequest(receipt messaging.MsgReceipt) error {
	reqcontent := receipt.Msg.Content.(map[string]interface{})
	code := reqcontent["code"].(string)
	status, indent := kernel.checkComplete(code)

	return receipt.Reply("is_complete_reply",
		isCompleteReply{
			Status: status,
			Indent: indent,
		},
	)
}

func handleShutdownRequest(receipt messaging.MsgReceipt) {
	content := receipt.Msg.Content.(map[string]interface{})
	restart := content["restart"].(bool)

	reply := shutdownReply{
		Restart: restart,
	}

	if err := receipt.Reply("shutdown_reply", reply); err != nil {
		log.Fatal(err)
	}

	log.Println("Shutting down in response to shutdown_request")
	os.Exit(0)
}
