package extension

import (
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"

	"github.com/alis-is/tezpay/cmd"
	"github.com/alis-is/tezpay/configuration"
)

func main() {
	s := rpc.NewServer()
	s.Register(Plugin{})
	s.ServeCodec(jsonrpc.NewServerCodec(rwCloser{os.Stdin, os.Stdout}))
}

// Plugin's methods become RPC calls.
type Plugin struct{}

func (Plugin) Hello() string { return "hello world!" }

// rwCloser just merges a ReadCloser and a WriteCloser into a ReadWriteCloser.
type rwCloser struct {
	io.ReadCloser
	io.WriteCloser
}

func (rw rwCloser) Close() error {
	err := rw.ReadCloser.Close()
	if err := rw.WriteCloser.Close(); err != nil {
		return err
	}
	return err
}

type Extension struct {
}

func (e *Extension) Init(options map[string]interface{}, configuration.RuntimeConfiguration) error {
	// create rpc server
	// register extension methods
	// start rpc server with jsonrpc codec
}



// function CreateExtensionInstance which starts extension from path, creates stdio pipes with jsonrpc codec over them and return extension instance
func CreateExtensionInstance(path string) (*Extension, error) {
	cmd := exec.Command(path)
	pw, pr, err := os.Pipe()
	pw2, pr2, err := os.Pipe()
	cmd.Stdin = pr
	cmd.Stdout = pw2

	// new jsonrpc client from stdin and stdout

	client := rpc.NewClientWithCodec(jsonrpc.NewClientCodec(rwCloser{pr2, pw}))
	client.Call()

	cmd.Execute()
}
