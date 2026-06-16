package messaging

import (
	"github.com/go-zeromq/zmq4"
	"github.com/gopherdata/gophernotes/internal/rendering"
)

func (receipt *MsgReceipt) SendResponse(socket zmq4.Socket, msg ComposedMsg) error {
	msgParts, err := msg.ToWireMsg(receipt.Sockets.Key)
	if err != nil {
		return err
	}

	var frames = make([][]byte, 0, len(receipt.Identities)+1+len(msgParts))
	frames = append(frames, receipt.Identities...)
	frames = append(frames, []byte("<IDS|MSG>"))
	frames = append(frames, msgParts...)

	err = socket.SendMulti(zmq4.NewMsgFrom(frames...))
	if err != nil {
		return err
	}

	return nil
}

func (receipt *MsgReceipt) Publish(msgType string, content interface{}) error {
	msg, err := NewMsg(msgType, receipt.Msg)
	if err != nil {
		return err
	}

	msg.Content = content
	return receipt.Sockets.IOPubSocket.RunWithSocket(func(iopub zmq4.Socket) error {
		return receipt.SendResponse(iopub, msg)
	})
}

func (receipt *MsgReceipt) Reply(msgType string, content interface{}) error {
	msg, err := NewMsg(msgType, receipt.Msg)
	if err != nil {
		return err
	}

	msg.Content = content
	return receipt.Sockets.ShellSocket.RunWithSocket(func(shell zmq4.Socket) error {
		return receipt.SendResponse(shell, msg)
	})
}

func (receipt *MsgReceipt) PublishKernelStatus(status string) error {
	return receipt.Publish("status",
		struct {
			ExecutionState string `json:"execution_state"`
		}{
			ExecutionState: status,
		},
	)
}

func (receipt *MsgReceipt) PublishExecutionInput(execCount int, code string) error {
	return receipt.Publish("execute_input",
		struct {
			ExecCount int    `json:"execution_count"`
			Code      string `json:"code"`
		}{
			ExecCount: execCount,
			Code:      code,
		},
	)
}

func (receipt *MsgReceipt) PublishExecutionResult(execCount int, data rendering.Data) error {
	return receipt.Publish("execute_result", struct {
		ExecCount int               `json:"execution_count"`
		Data      rendering.MIMEMap `json:"data"`
		Metadata  rendering.MIMEMap `json:"metadata"`
	}{
		ExecCount: execCount,
		Data:      data.Data,
		Metadata:  rendering.Ensure(data.Metadata),
	})
}

func (receipt *MsgReceipt) PublishExecutionError(err string, trace []string) error {
	return receipt.Publish("error",
		struct {
			Name  string   `json:"ename"`
			Value string   `json:"evalue"`
			Trace []string `json:"traceback"`
		}{
			Name:  "ERROR",
			Value: err,
			Trace: trace,
		},
	)
}

func (receipt *MsgReceipt) PublishDisplayData(data rendering.Data) error {
	return receipt.Publish("display_data", struct {
		Data      rendering.MIMEMap `json:"data"`
		Metadata  rendering.MIMEMap `json:"metadata"`
		Transient rendering.MIMEMap `json:"transient"`
	}{
		Data:      data.Data,
		Metadata:  rendering.Ensure(data.Metadata),
		Transient: rendering.Ensure(data.Transient),
	})
}

func (receipt *MsgReceipt) PublishWriteStream(stream string, data string) error {
	return receipt.Publish("stream",
		struct {
			Stream string `json:"name"`
			Data   string `json:"text"`
		}{
			Stream: stream,
			Data:   data,
		},
	)
}
