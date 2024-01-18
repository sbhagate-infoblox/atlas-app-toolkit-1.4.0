package gateway

import (
	"context"
	"net/http"
	"net/url"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/sbhagate-infoblox/atlas-app-toolkit-1.4.0/query"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// DefaultServerAddress is the standard gRPC server address that a REST
	// gateway will connect to.
	DefaultServerAddress = ":9090"
)

// Option is a functional option that modifies the REST gateway on
// initialization
type Option func(*gateway)

type registerFunc func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) (err error)

type gateway struct {
	serverAddress     string
	serverDialOptions []grpc.DialOption
	endpoints         map[string][]registerFunc
	mux               *http.ServeMux
	gatewayMuxOptions []runtime.ServeMuxOption
}

// ClientUnaryInterceptor parse collection operators and stores in corresponding message fields
func ClientUnaryInterceptor(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	raw, ok := Header(parentCtx, query_url)
	if ok {
		request, err := url.Parse(raw)
		if err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		vals := request.Query()
		// extracts "_order_by" parameters from request
		if v := vals.Get(sortQueryKey); v != "" {
			s, err := query.ParseSorting(v)
			if err != nil {
				return status.Error(codes.InvalidArgument, err.Error())
			}
			err = SetCollectionOps(req, s)
			if err != nil {
				return err
			}
		}
		// extracts "_fields" parameters from request
		if v := vals.Get(fieldsQueryKey); v != "" {
			fs := query.ParseFieldSelection(v)
			err := SetCollectionOps(req, fs)
			if err != nil {
				return err
			}
		}

		// extracts "_filter" parameters from request
		if v := vals.Get(filterQueryKey); v != "" {
			f, err := query.ParseFiltering(v)
			if err != nil {
				return status.Error(codes.InvalidArgument, err.Error())
			}

			err = SetCollectionOps(req, f)
			if err != nil {
				return err
			}
		}

		// extracts "_fts" parameters from request
		if v := vals.Get(searchQueryKey); v != "" {
			s := query.ParseSearching(v)
			err = SetCollectionOps(req, s)
			if err != nil {
				return err
			}
		}

		// extracts "_limit", "_offset",  "_page_token" parameters from request
		var p *query.Pagination
		l := vals.Get(limitQueryKey)
		o := vals.Get(offsetQueryKey)
		pt := vals.Get(pageTokenQueryKey)

		p, err = query.ParsePagination(l, o, pt)
		if err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		err = SetCollectionOps(req, p)
		if err != nil {
			return err
		}
	}
	return invoker(parentCtx, method, req, reply, cc, opts...)
}

// NewGateway creates a gRPC REST gateway with HTTP handlers that have been
// generated by the gRPC gateway protoc plugin
func NewGateway(options ...Option) (*http.ServeMux, error) {
	// configure gateway defaults
	g := gateway{
		serverAddress:     DefaultServerAddress,
		endpoints:         make(map[string][]registerFunc),
		serverDialOptions: []grpc.DialOption{grpc.WithInsecure(), grpc.WithUnaryInterceptor(ClientUnaryInterceptor)},
		mux:               http.NewServeMux(),
	}
	// apply functional options
	for _, opt := range options {
		opt(&g)
	}
	return g.registerEndpoints()
}

// registerEndpoints iterates through each prefix and registers its handlers
// to the REST gateway
func (g gateway) registerEndpoints() (*http.ServeMux, error) {
	for prefix, registers := range g.endpoints {
		gwmux := runtime.NewServeMux(
			append([]runtime.ServeMuxOption{runtime.WithErrorHandler(ProtoMessageErrorHandler),
				runtime.WithMetadata(MetadataAnnotator)}, g.gatewayMuxOptions...)...,
		)
		for _, register := range registers {
			if err := register(
				context.Background(), gwmux, g.serverAddress, g.serverDialOptions,
			); err != nil {
				return nil, err
			}
		}
		// strip prefix from testRequest URI, but leave the trailing "/"
		g.mux.Handle(prefix, http.StripPrefix(prefix[:len(prefix)-1], gwmux))
	}
	return g.mux, nil
}

// WithDialOptions assigns a list of gRPC dial options to the REST gateway
func WithDialOptions(options ...grpc.DialOption) Option {
	return func(g *gateway) {
		g.serverDialOptions = options
	}
}

// WithEndpointRegistration takes a group of HTTP handlers that have been
// generated by the gRPC gateway protoc plugin and registers them to the REST
// gateway with some prefix (e.g. www.website.com/prefix/endpoint)
func WithEndpointRegistration(prefix string, endpoints ...registerFunc) Option {
	return func(g *gateway) {
		g.endpoints[prefix] = append(g.endpoints[prefix], endpoints...)
	}
}

// WithServerAddress determines what address the gateway will connect to. By
// default, the gateway will connect to 0.0.0.0:9090
func WithServerAddress(address string) Option {
	return func(g *gateway) {
		g.serverAddress = address
	}
}

// WithMux will use the given http.ServeMux to register the gateway endpoints.
func WithMux(mux *http.ServeMux) Option {
	return func(g *gateway) {
		g.mux = mux
	}
}

// WithGatewayOptions allows for additional gateway ServeMuxOptions beyond the
// default ProtoMessageErrorHandler and MetadataAnnotator from this package
func WithGatewayOptions(opt ...runtime.ServeMuxOption) Option {
	return func(g *gateway) {
		g.gatewayMuxOptions = append(g.gatewayMuxOptions, opt...)
	}
}
