package extension

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"time"

	"github.com/alis-is/jsonrpc2/endpoints"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/utils"
	"github.com/sirupsen/logrus"
)

type rwCloser struct {
	io.ReadCloser
	io.WriteCloser
}

func (rw rwCloser) Close() error {
	return errors.Join(rw.WriteCloser.Close(), rw.ReadCloser.Close())
}

type StdioExtension struct {
	ExtensionBase
	ctx    context.Context
	loaded bool
}

func newStdioExtension(ctx context.Context, def common.ExtensionDefinition) Extension {
	return &StdioExtension{
		ExtensionBase: ExtensionBase{
			definition: def,
		},
		ctx: ctx,
	}
}
func (e *StdioExtension) Load() error {
	if e.IsLoaded() && e.endpoint != nil {
		// cleanup old endpoint
		return e.endpoint.Close()
	}
	args, err := utils.SplitStringToArgs(e.GetDefinition().Command)
	if err != nil {
		return fmt.Errorf("invalid command: %s", err)
	}
	if len(args) == 0 {
		return errors.New("no command specified")
	}
	cmd := exec.Command(args[0], args[1:]...)
	pw, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	pr, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	objStream := endpoints.NewPlainObjectStream(rwCloser{pr, pw})
	streamEndpoint := endpoints.NewStreamEndpoint(e.ctx, objStream)
	streamEndpoint.UseLogger(logrus.StandardLogger())
	if err := cmd.Start(); err != nil {
		return err
	}
	time.Sleep(time.Duration(e.definition.WaitForStart) * time.Second)
	// init extension

	e.endpoint = streamEndpoint
	e.loaded = true
	return nil
}

func (e *StdioExtension) IsLoaded() bool {
	if e.endpoint == nil {
		return false
	}
	err := endpoints.Notify[interface{}](e.ctx, e.endpoint, "tp.healthcheck", nil)
	return e.loaded && err == nil
}

func (e *StdioExtension) Close() error {
	e.loaded = false
	if e.endpoint == nil {
		return nil
	}
	endpoints.Notify[interface{}](e.ctx, e.endpoint, "close", nil)
	return errors.Join(e.endpoint.Close())
}

type TcpExtension struct {
	ExtensionBase
	ctx  context.Context
	conn net.Conn
}

func newTcpExtension(ctx context.Context, def common.ExtensionDefinition) Extension {
	return &TcpExtension{
		ExtensionBase: ExtensionBase{
			definition: def,
		},
		ctx: ctx,
	}
}
func (e *TcpExtension) Load() error {
	if e.IsLoaded() && e.endpoint != nil {
		// cleanup old endpoint
		return e.endpoint.Close()
	}
	cmd := e.GetDefinition().Command
	if cmd != "" {
		args, err := utils.SplitStringToArgs(e.GetDefinition().Command)
		if err != nil {
			return fmt.Errorf("invalid command: %s", err)
		}
		if len(args) != 0 {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Start()
			if err := cmd.Start(); err != nil {
				return err
			}
			time.Sleep(time.Duration(e.definition.WaitForStart) * time.Second)
		}
	}
	conn, err := net.Dial("tcp", e.GetDefinition().Url)
	if err != nil {
		return err
	}
	objStream := endpoints.NewPlainObjectStream(conn)
	streamEndpoint := endpoints.NewStreamEndpoint(e.ctx, objStream)
	streamEndpoint.UseLogger(logrus.StandardLogger())
	e.endpoint = streamEndpoint
	e.loaded = true
	return nil
}

func (e *TcpExtension) IsLoaded() bool {
	if e.endpoint == nil {
		return false
	}
	err := endpoints.Notify[interface{}](e.ctx, e.endpoint, "tp.healthcheck", nil)
	return e.loaded && err == nil
}

func (e *TcpExtension) Close() error {
	e.loaded = false
	endpoints.Notify[interface{}](e.ctx, e.endpoint, "close", nil)
	err := e.endpoint.Close()
	if err != nil {
		return errors.Join(err, e.conn.Close())
	}
	return e.conn.Close()
}
