package kernel

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"

	interp "github.com/pjgraczyk/gomacro/fast"
	"github.com/pjgraczyk/gomacro/xreflect"
	"github.com/go-zeromq/zmq4"

	"github.com/gopherdata/gophernotes/internal/messaging"
	"github.com/gopherdata/gophernotes/internal/rendering"
)

const (
	Version = "1.0.0"

	statusStarting = "starting"
	statusBusy     = "busy"
	statusIdle     = "idle"
)

var ExecCounter int

type Kernel struct {
	ir      *interp.Interp
	display *interp.Import
	render  map[string]xreflect.Type
}

func RunKernel(connectionFile string) {
	ir := interp.New()

	ir.Comp.Stdout = io.Discard
	ir.Comp.Stderr = io.Discard

	display := importPackage(ir, "display", "display")

	ir.DeclVar("Display", nil, rendering.StubDisplay)

	var connInfo messaging.ConnectionInfo
	connData, err := os.ReadFile(connectionFile)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal(connData, &connInfo); err != nil {
		log.Fatal(err)
	}

	sockets, err := prepareSockets(connInfo)
	if err != nil {
		log.Fatal(err)
	}

	startHeartbeat(sockets.HBSocket, &sync.WaitGroup{})

	type msgType struct {
		Msg zmq4.Msg
		Err error
	}

	var (
		shell = make(chan msgType)
		stdin = make(chan msgType)
		ctl   = make(chan msgType)
		quit  = make(chan int)
	)

	defer close(quit)
	poll := func(msgs chan msgType, sck zmq4.Socket) {
		defer close(msgs)
		for {
			msg, err := sck.Recv()
			select {
			case msgs <- msgType{Msg: msg, Err: err}:
			case <-quit:
				return
			}
		}
	}

	go poll(shell, sockets.ShellSocket.Socket)
	go poll(stdin, sockets.StdinSocket.Socket)
	go poll(ctl, sockets.ControlSocket.Socket)

	kernel := Kernel{ir, display, nil}
	kernel.initRenderers()

	for {
		select {
		case v := <-shell:
			if v.Err != nil {
				log.Println(v.Err)
				continue
			}

			msg, ids, err := messaging.WireMsgToComposedMsg(v.Msg.Frames, sockets.Key)
			if err != nil {
				log.Println(err)
				return
			}

			kernel.handleShellMsg(messaging.MsgReceipt{Msg: msg, Identities: ids, Sockets: sockets})

		case <-stdin:
			continue

		case v := <-ctl:
			if v.Err != nil {
				log.Println(v.Err)
				return
			}

			msg, ids, err := messaging.WireMsgToComposedMsg(v.Msg.Frames, sockets.Key)
			if err != nil {
				log.Println(err)
				return
			}

			kernel.handleShellMsg(messaging.MsgReceipt{Msg: msg, Identities: ids, Sockets: sockets})
		}
	}
}

func importPackage(ir *interp.Interp, path string, alias string) *interp.Import {
	packages, err := ir.ImportPackagesOrError(
		map[string]interp.PackageName{
			path: interp.PackageName(alias),
		})
	if err != nil {
		log.Print(err)
	}
	return packages[path]
}
