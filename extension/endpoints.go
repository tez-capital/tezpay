package extension

import (
	"context"
	"io"

	rpc "github.com/alis-is/jsonrpc2"
	"github.com/tez-capital/tezpay/constants"
)

/*
All references to rpc are redirected from here to the endpoints package.
This is done so we can adjust calls (inject call prefix) while mainting consistent behavior across the extension package.
"github.com/alis-is/jsonrpc2" should not be used directly
*/

type EndpointClient = rpc.EndpointClient

func Request[TParams rpc.Params, TResult rpc.Result](ctx context.Context, c EndpointClient, method string, params TParams) (*rpc.Response[TResult], error) {
	return rpc.Request[TParams, TResult](ctx, c, constants.EXTENSION_CALL_PREFIX+method, params)
}

func Notify[TParams rpc.Params](ctx context.Context, c EndpointClient, method string, params TParams) error {
	return rpc.Notify(ctx, c, constants.EXTENSION_CALL_PREFIX+method, params)
}

func RequestTo[TParams rpc.Params, TResult rpc.Result](ctx context.Context, c EndpointClient, method string, params TParams, result *rpc.Response[TResult]) error {
	return rpc.RequestTo(ctx, c, constants.EXTENSION_CALL_PREFIX+method, params, result)
}

func RegisterEndpointMethod[TParam rpc.Params, TResult rpc.Result](c rpc.EndpointServer, method string, handler rpc.RpcMethod[TParam, TResult]) {
	rpc.RegisterEndpointMethod(c, constants.EXTENSION_CALL_PREFIX+method, handler)
}

func NewPlainObjectStream(rw io.ReadWriteCloser) rpc.ObjectStream {
	return rpc.NewPlainObjectStream(rw)
}

func NewStreamEndpoint(ctx context.Context, stream rpc.ObjectStream) *rpc.StreamEndpoint {
	return rpc.NewStreamEndpoint(ctx, stream)
}
