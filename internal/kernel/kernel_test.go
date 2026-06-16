package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"github.com/gopherdata/gophernotes/internal/messaging"
)

const (
	connectionFile = "testdata/connection_file.json"
	sessionID      = "ba65a05c-106a-4799-9a94-7f5631bbe216"

	testFailure = "\u2717"
	testSuccess = "\u2713"
)

var (
	connectionKey string
	transport     string
	ip            string
	shellPort     int
	iopubPort     int
)

func TestMain(m *testing.M) {
	os.Exit(runTest(m))
}

func runTest(m *testing.M) int {
	var connInfo messaging.ConnectionInfo

	connData, err := os.ReadFile(connectionFile)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal(connData, &connInfo); err != nil {
		log.Fatal(err)
	}

	connectionKey = connInfo.Key
	transport = connInfo.Transport
	ip = connInfo.IP
	shellPort = connInfo.ShellPort
	iopubPort = connInfo.IOPubPort

	go RunKernel(connectionFile)

	return m.Run()
}

func TestEvaluate(t *testing.T) {
	cases := []struct {
		Input  []string
		Output string
	}{
		{[]string{
			"a := 1",
			"a",
		}, "1"},
		{[]string{
			"a = 2",
			"a + 3",
		}, "5"},
		{[]string{
			"func myFunc(x int) int {",
			"    return x+1",
			"}",
			"myFunc(1)",
		}, "2"},
		{[]string{
			"b := myFunc(1)",
		}, ""},
		{[]string{
			"type Rect struct {",
			"    Width, Height int",
			"}",
			"Rect{10, 30}",
		}, "{10 30}"},
		{[]string{
			"type Rect struct {",
			"    Width, Height int",
			"}",
			"&Rect{10, 30}",
		}, "&{10 30}"},
		{[]string{
			"func a(b int) (int, int) {",
			"    return 2 + b, b",
			"}",
			"a(10)",
		}, "12 10"},
		{[]string{
			`import "errors"`,
			"func a() (interface{}, error) {",
			`    return nil, errors.New("To err is human")`,
			"}",
			"a()",
		}, "<nil> To err is human"},
		{[]string{
			`c := []string{"gophernotes", "is", "super", "bad"}`,
			"c[:3]",
		}, "[gophernotes is super]"},
		{[]string{
			"m := map[string]int{",
			`    "a": 10,`,
			`    "c": 30,`,
			"}",
			`m["c"]`,
		}, "30 true"},
		{[]string{
			"if 1 < 2 {",
			"    3",
			"}",
		}, ""},
		{[]string{
			"d := 10",
			"d++",
		}, ""},
		{[]string{
			"out := make(chan int)",
			"go func() {",
			"    out <- 123",
			"}()",
			"<-out",
		}, "123 true"},
	}

	t.Logf("Should be able to evaluate valid code in notebook cells.")

	for k, tc := range cases {
		t.Logf("  Evaluating code snippet %d/%d.", k+1, len(cases))

		result := testEvaluate(t, strings.Join(tc.Input, "\n"))

		if result != tc.Output {
			t.Errorf("\t%s Test case produced unexpected results.", testFailure)
			continue
		}
		t.Logf("\t%s Should return the correct cell output.", testSuccess)
	}
}

func TestPanicGeneratesError(t *testing.T) {
	client, closeClient := newTestJupyterClient(t)
	defer closeClient()

	content, pub := client.executeCode(t, `panic("error")`)

	status := getString(t, "content", content, "status")

	if status != "error" {
		t.Fatalf("\t%s Execution did not raise expected error", testFailure)
	}

	var foundPublishedError bool
	for _, pubMsg := range pub {
		if pubMsg.Header.MsgType == "error" {
			foundPublishedError = true
			break
		}
	}

	if !foundPublishedError {
		t.Fatalf("\t%s Execution did not publish an expected \"error\" message", testFailure)
	}
}

func TestPrintStdout(t *testing.T) {
	cases := []struct {
		Input  []string
		Output []string
	}{
		{[]string{
			`import "fmt"`,
			"a := 1",
			"fmt.Println(a)",
		}, []string{"1\n"}},
		{[]string{
			"a = 2",
			"fmt.Print(a)",
		}, []string{"2"}},
		{[]string{
			`import "os"`,
			`os.Stdout.WriteString("3")`,
		}, []string{"3"}},
		{[]string{
			`fmt.Fprintf(os.Stdout, "%d\n", 4)`,
		}, []string{"4\n"}},
		{[]string{
			`import "time"`,
			"for i := 0; i < 3; i++ {",
			"    fmt.Println(i)",
			"    time.Sleep(500 * time.Millisecond)",
			"}",
		}, []string{"0\n", "1\n", "2\n"}},
	}

	t.Logf("Should produce stdout stream messages when writing to stdout")

cases:
	for k, tc := range cases {
		t.Logf("  Evaluating code snippet %d/%d.", k+1, len(cases))

		stdout, _ := testOutputStream(t, strings.Join(tc.Input, "\n"))

		if len(stdout) != len(tc.Output) {
			t.Errorf("\t%s Test case expected %d message(s) on stdout but got %d.", testFailure, len(tc.Output), len(stdout))
			continue
		}
		for i, expected := range tc.Output {
			if stdout[i] != expected {
				t.Errorf("\t%s Test case returned unexpected messages on stdout.", testFailure)
				continue cases
			}
		}
		t.Logf("\t%s Returned the expected messages on stdout.", testSuccess)
	}
}

func TestPrintStderr(t *testing.T) {
	cases := []struct {
		Input  []string
		Output []string
	}{
		{[]string{
			`import "fmt"`,
			`import "os"`,
			"a := 1",
			"fmt.Fprintln(os.Stderr, a)",
		}, []string{"1\n"}},
		{[]string{
			`os.Stderr.WriteString("2")`,
		}, []string{"2"}},
		{[]string{
			`import "time"`,
			"for i := 0; i < 3; i++ {",
			"    fmt.Fprintln(os.Stderr, i)",
			"    time.Sleep(500 * time.Millisecond)",
			"}",
		}, []string{"0\n", "1\n", "2\n"}},
	}

	t.Logf("Should produce stderr stream messages when writing to stderr")

cases:
	for k, tc := range cases {
		t.Logf("  Evaluating code snippet %d/%d.", k+1, len(cases))

		_, stderr := testOutputStream(t, strings.Join(tc.Input, "\n"))

		if len(stderr) != len(tc.Output) {
			t.Errorf("\t%s Test case expected %d message(s) on stderr but got %d.", testFailure, len(tc.Output), len(stderr))
			continue
		}
		for i, expected := range tc.Output {
			if stderr[i] != expected {
				t.Errorf("\t%s Test case returned unexpected messages on stderr.", testFailure)
				continue cases
			}
		}
		t.Logf("\t%s Returned the expected messages on stderr.", testSuccess)
	}
}

type testJupyterClient struct {
	shellSocket zmq4.Socket
	ioSocket    zmq4.Socket
}

func newTestJupyterClient(t *testing.T) (testJupyterClient, func()) {
	t.Helper()

	var (
		err       error
		ctx       = context.Background()
		addrShell = fmt.Sprintf("%s://%s:%d", transport, ip, shellPort)
		addrIO    = fmt.Sprintf("%s://%s:%d", transport, ip, iopubPort)
	)

	shell := zmq4.NewReq(ctx)
	if err = shell.Dial(addrShell); err != nil {
		t.Fatalf("\t%s shell.Connect: %s", testFailure, err)
	}

	iopub := zmq4.NewSub(ctx)
	if err = iopub.Dial(addrIO); err != nil {
		t.Fatalf("\t%s iopub.Connect: %s", testFailure, err)
	}

	if err = iopub.SetOption(zmq4.OptionSubscribe, ""); err != nil {
		t.Fatalf("\t%s iopub.SetSubscribe: %s", testFailure, err)
	}

	time.Sleep(1 * time.Second)

	return testJupyterClient{shell, iopub}, func() {
		if err := shell.Close(); err != nil {
			t.Errorf("\t%s shell.Close: %s", testFailure, err)
		}
		if err = iopub.Close(); err != nil {
			t.Errorf("\t%s iopub.Close: %s", testFailure, err)
		}
	}
}

func (client *testJupyterClient) sendShellRequest(t *testing.T, request messaging.ComposedMsg) {
	t.Helper()

	var (
		frames [][]byte
		err    error
	)

	frames = append(frames, []byte("<IDS|MSG>"))

	reqMsgParts, err := request.ToWireMsg([]byte(connectionKey))
	if err != nil {
		t.Fatalf("\t%s request.ToWireMsg: %s", testFailure, err)
	}
	frames = append(frames, reqMsgParts...)

	if err = client.shellSocket.SendMulti(zmq4.NewMsgFrom(frames...)); err != nil {
		t.Fatalf("\t%s shellSocket.SendMessage: %s", testFailure, err)
	}
}

func (client *testJupyterClient) recvShellReply(t *testing.T, timeout time.Duration) messaging.ComposedMsg {
	t.Helper()

	type result struct {
		msg messaging.ComposedMsg
		err error
	}
	ch := make(chan result, 1)

	go func() {
		repMsgParts, err := client.shellSocket.Recv()
		if err != nil {
			ch <- result{err: fmt.Errorf("Shell socket RecvMessageBytes: %w", err)}
			return
		}

		msgParsed, _, err := messaging.WireMsgToComposedMsg(repMsgParts.Frames, []byte(connectionKey))
		if err != nil {
			ch <- result{err: fmt.Errorf("Could not parse wire message: %w", err)}
			return
		}

		ch <- result{msg: msgParsed}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("\t%s %s", testFailure, r.err)
		}
		return r.msg
	case <-time.After(timeout):
		t.Fatalf("\t%s recvShellReply timed out", testFailure)
	}

	return messaging.ComposedMsg{}
}

func (client *testJupyterClient) recvIOSub(t *testing.T, timeout time.Duration) messaging.ComposedMsg {
	t.Helper()

	type result struct {
		msg messaging.ComposedMsg
		err error
	}
	ch := make(chan result, 1)

	go func() {
		repMsgParts, err := client.ioSocket.Recv()
		if err != nil {
			ch <- result{err: fmt.Errorf("IOPub socket RecvMessageBytes: %w", err)}
			return
		}

		msgParsed, _, err := messaging.WireMsgToComposedMsg(repMsgParts.Frames, []byte(connectionKey))
		if err != nil {
			ch <- result{err: fmt.Errorf("Could not parse wire message: %w", err)}
			return
		}

		ch <- result{msg: msgParsed}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("\t%s %s", testFailure, r.err)
		}
		return r.msg
	case <-time.After(timeout):
		t.Fatalf("\t%s recvIOSub timed out", testFailure)
	}

	return messaging.ComposedMsg{}
}

func (client *testJupyterClient) performJupyterRequest(t *testing.T, request messaging.ComposedMsg, timeout time.Duration) (messaging.ComposedMsg, []messaging.ComposedMsg) {
	t.Helper()

	client.sendShellRequest(t, request)
	reply := client.recvShellReply(t, timeout)

	subMsg := client.recvIOSub(t, 1*time.Second)
	assertMsgTypeEquals(t, subMsg, "status")

	subData := getMsgContentAsJSONObject(t, subMsg)
	execState := getString(t, "content", subData, "execution_state")

	if execState != statusBusy {
		t.Fatalf("\t%s Expected a 'busy' status message but got '%s'", testFailure, execState)
	}

	var pub []messaging.ComposedMsg

	for {
		subMsg = client.recvIOSub(t, 100*time.Millisecond)

		if subMsg.Header.MsgType == "status" {
			subData = getMsgContentAsJSONObject(t, subMsg)
			execState = getString(t, "content", subData, "execution_state")

			if execState != statusIdle {
				t.Fatalf("\t%s Expected a 'idle' status message but got '%s'", testFailure, execState)
			}

			break
		}

		pub = append(pub, subMsg)
	}

	return reply, pub
}

func (client *testJupyterClient) executeCode(t *testing.T, code string) (map[string]interface{}, []messaging.ComposedMsg) {
	t.Helper()

	request, err := messaging.NewMsg("execute_request", messaging.ComposedMsg{})
	if err != nil {
		t.Fatalf("\t%s NewMsg: %s", testFailure, err)
	}

	request.Header.Session = sessionID
	request.Header.Username = "KernelTester"

	request.Metadata = make(map[string]interface{})

	content := make(map[string]interface{})
	content["code"] = code
	content["silent"] = false
	request.Content = content

	reply, pub := client.performJupyterRequest(t, request, 10*time.Second)

	assertMsgTypeEquals(t, reply, "execute_reply")
	content = getMsgContentAsJSONObject(t, reply)

	return content, pub
}

func testEvaluate(t *testing.T, codeIn string) string {
	client, closeClient := newTestJupyterClient(t)
	defer closeClient()

	content, pub := client.executeCode(t, codeIn)

	status := getString(t, "content", content, "status")

	if status != "ok" {
		t.Fatalf("\t%s Execution encountered error [%s]: %s", testFailure, content["ename"], content["evalue"])
	}

	for _, pubMsg := range pub {
		if pubMsg.Header.MsgType == "execute_result" {
			content = getMsgContentAsJSONObject(t, pubMsg)

			bundledMIMEData := getJSONObject(t, "content", content, "data")
			textRep := getString(t, `content["data"]`, bundledMIMEData, "text/plain")

			return textRep
		}
	}

	return ""
}

func assertMsgTypeEquals(t *testing.T, msg messaging.ComposedMsg, expectedType string) {
	t.Helper()

	if msg.Header.MsgType != expectedType {
		t.Fatalf("\t%s Expected message of type '%s' but was '%s'", testFailure, expectedType, msg.Header.MsgType)
	}
}

func getMsgContentAsJSONObject(t *testing.T, msg messaging.ComposedMsg) map[string]interface{} {
	t.Helper()

	content, ok := msg.Content.(map[string]interface{})
	if !ok {
		t.Fatalf("\t%s Message content is not a JSON object", testFailure)
	}

	return content
}

func getString(t *testing.T, jsonObjectName string, content map[string]interface{}, key string) string {
	t.Helper()

	raw, ok := content[key]
	if !ok {
		t.Fatalf("\t%s %s[\"%s\"] field not present", testFailure, jsonObjectName, key)
	}

	value, ok := raw.(string)
	if !ok {
		t.Fatalf("\t%s %s[\"%s\"] is not a string", testFailure, jsonObjectName, key)
	}

	return value
}

func getJSONObject(t *testing.T, jsonObjectName string, content map[string]interface{}, key string) map[string]interface{} {
	t.Helper()

	raw, ok := content[key]
	if !ok {
		t.Fatalf("\t%s %s[\"%s\"] field not present", testFailure, jsonObjectName, key)
	}

	value, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("\t%s %s[\"%s\"] is not a JSON object", testFailure, jsonObjectName, key)
	}

	return value
}

func testOutputStream(t *testing.T, codeIn string) ([]string, []string) {
	t.Helper()

	client, closeClient := newTestJupyterClient(t)
	defer closeClient()

	_, pub := client.executeCode(t, codeIn)

	var stdout, stderr []string
	for _, pubMsg := range pub {
		if pubMsg.Header.MsgType == "stream" {
			content := getMsgContentAsJSONObject(t, pubMsg)
			streamType := getString(t, "content", content, "name")
			streamData := getString(t, "content", content, "text")

			switch streamType {
			case messaging.StreamStdout:
				stdout = append(stdout, streamData)
			case messaging.StreamStderr:
				stderr = append(stderr, streamData)
			default:
				t.Fatalf("Unknown stream type '%s'", streamType)
			}
		}
	}

	return stdout, stderr
}
