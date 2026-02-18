package server

import (
	"context"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	api "github.com/platform9/vjailbreak/pkg/vpwned/api/proto/v1/service"
	"github.com/platform9/vjailbreak/pkg/vpwned/openapiv3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

var grpcServer *grpc.Server
var httpServer *http.Server

// sets up the swagger UI server for Rest API's
func openAPIServer(mux *http.ServeMux, dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "swagger.json") {
			logrus.Errorf("path not found: %s", r.URL.Path)
			static, err := fs.Sub(openapiv3.OpenAPI, "dist")
			if err != nil {
				logrus.Errorf("cannot embed openAPI, err: %v", err)
			}
			http.StripPrefix("/swagger/", http.FileServer(http.FS(static))).ServeHTTP(w, r)
			//http.FileServer(http.FS(static)).ServeHTTP(w, r)
			return
		}
		tp := strings.TrimPrefix(r.URL.Path, "/swagger/")
		tp = path.Join(dir, tp)
		logrus.Infof("Trimmed Path: %s", tp)
		var readE error
		var data []byte
		if _, err := os.Stat(tp); err == nil {
			data, readE = os.ReadFile(tp)
			if readE != nil {
				logrus.Errorf("error reading: %s, err: %v", tp, readE)
			}
		} else {
			logrus.Errorf("file %s, encountered err: %v", tp, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Write(data)
	} //end of return
} //end of openAPIServer

// setup grpc server
func startgRPCServer(ctx context.Context, network, port string) error {
	grpcServer = grpc.NewServer()

	k8sClient, err := CreateInClusterClient()
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client for grpc server")
	}

	//Register all services here
	//TODO: Register proto servers here.
	api.RegisterVersionServer(grpcServer, &VpwnedVersion{})
	api.RegisterVCenterServer(grpcServer, &targetVcenterGRPC{})
	api.RegisterBMProviderServer(grpcServer, &providersGRPC{})
	api.RegisterVailbreakProxyServer(grpcServer, &vjailbreakProxy{K8sClient: k8sClient})
	api.RegisterStorageArrayServer(grpcServer, &storageArrayGRPC{})
	reflection.Register(grpcServer)
	connection, err := net.Listen(network, port)
	if err != nil {
		logrus.Errorf("cannot listen on port: %s:%s, err: %v", network, port, err)
	}

	//start the server in a go routine
	//so that we can return from here
	go func() {
		defer grpcServer.GracefulStop()
		<-ctx.Done()
	}()
	return grpcServer.Serve(connection)
}

// TODO: write this
func gRPCErrHandler(ctx context.Context, mux *runtime.ServeMux, m runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	for _, opts := range mux.GetForwardResponseOptions() {
		if err := opts(ctx, w, nil); err != nil {
			logrus.Error(err)
			return
		}
	}
	runtime.DefaultHTTPErrorHandler(ctx, mux, m, w, r, err)
}

func APILogger(fwd http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logrus.Info(r.Method, r.RemoteAddr, r.RequestURI)
		fwd.ServeHTTP(w, r)
	})
}

func getHTTPServer(ctx context.Context, port, grpcSocket string) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	//TODO: Move this path to a direct path in the /tmp or a path in the container
	// or take it via config or env variable
	mux.HandleFunc("/swagger/", openAPIServer(mux, "/opt/platform9/vpwned/openapiv3/dist/"))

	// Register VDDK handlers first with specific paths
	mux.HandleFunc("/vpw/v1/vddk/upload", HandleVDDKUpload)
	mux.HandleFunc("/vpw/v1/vddk/status", HandleVDDKStatus)

	//gatewayMuxer
	gatewayMuxer := runtime.NewServeMux() //runtime.WithErrorHandler(gRPCErrHandler))
	option := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// ctx, muxer, "127.0.0.1:3000", option
	if err := api.RegisterVersionHandlerFromEndpoint(ctx, gatewayMuxer, grpcSocket, option); err != nil {
		logrus.Errorf("cannot start handler for version")
	}
	// Register MachineInfo service
	if err := api.RegisterBMProviderHandlerFromEndpoint(ctx, gatewayMuxer, grpcSocket, option); err != nil {
		logrus.Errorf("cannot start handler for BMProvider")
	}
	// Register VCenter service
	if err := api.RegisterVCenterHandlerFromEndpoint(ctx, gatewayMuxer, grpcSocket, option); err != nil {
		logrus.Errorf("cannot start handler for VCenter")
	}
	// Register VJailbreakProxy service
	if err := api.RegisterVailbreakProxyHandlerFromEndpoint(ctx, gatewayMuxer, grpcSocket, option); err != nil {
		logrus.Errorf("cannot start handler for VailbreakProxy")
	}
	// Register StorageArray service
	if err := api.RegisterStorageArrayHandlerFromEndpoint(ctx, gatewayMuxer, grpcSocket, option); err != nil {
		logrus.Errorf("cannot start handler for StorageArray")
	}

	// Wrap gatewayMuxer to handle all other routes
	mux.HandleFunc("/vpw/", func(w http.ResponseWriter, r *http.Request) {
		// Skip VDDK endpoints - they're already registered
		if r.URL.Path == "/vpw/v1/vddk/upload" {
			HandleVDDKUpload(w, r)
			return
		}
		if r.URL.Path == "/vpw/v1/vddk/status" {
			HandleVDDKStatus(w, r)
			return
		}
		APILogger(gatewayMuxer).ServeHTTP(w, r)
	})

	return mux, nil
}

func StartServer(host, port, apiPort, apiHost string) error {
	ctx := context.Background()
	ctx, cncl := context.WithCancel(ctx)
	defer cncl()

	go func() {
		if err := startgRPCServer(ctx, "tcp", host+":"+port); err != nil {
			logrus.Error("cannot start grpc server", err)
		}
	}()
	logrus.Info("gRPC server started at:", host, ":", port)
	mux, err := getHTTPServer(ctx, apiHost+":"+apiPort, host+":"+port)
	if err != nil {
		logrus.Errorf("cannot start rest server: %v", err)
		return err
	}
	logrus.Info("starting http server.....")
	httpServer = &http.Server{
		Addr:           apiHost + ":" + apiPort,
		Handler:        mux,
		MaxHeaderBytes: 1 << 20, // 1 MB for headers
	}
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		logrus.Error("cannot start http server", err)
		return err
	}
	logrus.Info("Http server started at:", apiHost, ":", apiPort)
	return nil
}

func Shutdown(ctx context.Context) error {
	return httpServer.Shutdown(ctx)
}
