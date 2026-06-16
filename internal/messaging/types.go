package messaging

import (
	"io"
	"sync"

	"github.com/go-zeromq/zmq4"
)

const (
	StreamStdout = "stdout"
	StreamStderr = "stderr"
)

type ConnectionInfo struct {
	SignatureScheme string `json:"signature_scheme"`
	Transport       string `json:"transport"`
	StdinPort       int    `json:"stdin_port"`
	ControlPort     int    `json:"control_port"`
	IOPubPort       int    `json:"iopub_port"`
	HBPort          int    `json:"hb_port"`
	ShellPort       int    `json:"shell_port"`
	Key             string `json:"key"`
	IP              string `json:"ip"`
}

type MsgHeader struct {
	MsgID           string `json:"msg_id"`
	Username        string `json:"username"`
	Session         string `json:"session"`
	MsgType         string `json:"msg_type"`
	ProtocolVersion string `json:"version"`
	Timestamp       string `json:"date"`
}

type ComposedMsg struct {
	Header       MsgHeader
	ParentHeader MsgHeader
	Metadata     map[string]interface{}
	Content      interface{}
}

type MsgReceipt struct {
	Msg        ComposedMsg
	Identities [][]byte
	Sockets    SocketGroup
}

type Socket struct {
	Socket zmq4.Socket
	Lock   *sync.Mutex
}

type SocketGroup struct {
	ShellSocket   Socket
	ControlSocket Socket
	StdinSocket   Socket
	IOPubSocket   Socket
	HBSocket      Socket
	Key           []byte
}

type InvalidSignatureError struct{}

func (e *InvalidSignatureError) Error() string {
	return "A message had an invalid signature"
}

type OutErr struct {
	Out io.Writer
	Err io.Writer
}

type JupyterStreamWriter struct {
	Stream  string
	Receipt *MsgReceipt
}

func (writer *JupyterStreamWriter) Write(p []byte) (int, error) {
	data := string(p)
	n := len(p)

	if err := writer.Receipt.PublishWriteStream(writer.Stream, data); err != nil {
		return 0, err
	}

	return n, nil
}

func (s *Socket) RunWithSocket(run func(socket zmq4.Socket) error) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	return run(s.Socket)
}
