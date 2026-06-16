package kernel

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-zeromq/zmq4"

	"github.com/gopherdata/gophernotes/internal/messaging"
)

func prepareSockets(connInfo messaging.ConnectionInfo) (messaging.SocketGroup, error) {
	var (
		sg  messaging.SocketGroup
		err error
		ctx = context.Background()
	)

	sg.ShellSocket.Socket = zmq4.NewRouter(ctx)
	sg.ShellSocket.Lock = &sync.Mutex{}

	sg.ControlSocket.Socket = zmq4.NewRouter(ctx)
	sg.ControlSocket.Lock = &sync.Mutex{}

	sg.StdinSocket.Socket = zmq4.NewRouter(ctx)
	sg.StdinSocket.Lock = &sync.Mutex{}

	sg.IOPubSocket.Socket = zmq4.NewPub(ctx)
	sg.IOPubSocket.Lock = &sync.Mutex{}

	sg.HBSocket.Socket = zmq4.NewRep(ctx)
	sg.HBSocket.Lock = &sync.Mutex{}

	address := fmt.Sprintf("%v://%v:%%v", connInfo.Transport, connInfo.IP)
	err = sg.ShellSocket.Socket.Listen(fmt.Sprintf(address, connInfo.ShellPort))
	if err != nil {
		return sg, fmt.Errorf("could not listen on shell-socket: %w", err)
	}

	err = sg.ControlSocket.Socket.Listen(fmt.Sprintf(address, connInfo.ControlPort))
	if err != nil {
		return sg, fmt.Errorf("could not listen on control-socket: %w", err)
	}

	err = sg.StdinSocket.Socket.Listen(fmt.Sprintf(address, connInfo.StdinPort))
	if err != nil {
		return sg, fmt.Errorf("could not listen on stdin-socket: %w", err)
	}

	err = sg.IOPubSocket.Socket.Listen(fmt.Sprintf(address, connInfo.IOPubPort))
	if err != nil {
		return sg, fmt.Errorf("could not listen on iopub-socket: %w", err)
	}

	err = sg.HBSocket.Socket.Listen(fmt.Sprintf(address, connInfo.HBPort))
	if err != nil {
		return sg, fmt.Errorf("could not listen on hbeat-socket: %w", err)
	}

	sg.Key = []byte(connInfo.Key)

	return sg, nil
}

func startHeartbeat(hbSocket messaging.Socket, wg *sync.WaitGroup) (shutdown chan struct{}) {
	quit := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()

		type msgType struct {
			Msg zmq4.Msg
			Err error
		}

		msgs := make(chan msgType)

		go func() {
			defer close(msgs)
			for {
				msg, err := hbSocket.Socket.Recv()
				select {
				case msgs <- msgType{msg, err}:
				case <-quit:
					return
				}
			}
		}()

		timeout := time.NewTimer(500 * time.Second)
		defer timeout.Stop()

		for {
			timeout.Reset(500 * time.Second)
			select {
			case <-quit:
				return
			case <-timeout.C:
				continue
			case v := <-msgs:
				hbSocket.RunWithSocket(func(echo zmq4.Socket) error {
					if v.Err != nil {
						log.Fatalf("Error reading heartbeat ping bytes: %v\n", v.Err)
						return v.Err
					}

					if err := echo.Send(v.Msg); err != nil {
						log.Printf("Error sending heartbeat pong bytes: %b\n", err)
						return err
					}

					return nil
				})
			}
		}
	}()

	return quit
}
