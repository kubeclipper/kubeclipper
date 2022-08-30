package nodestatus

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

const (
	unixProtocol = "unix"
)

func ContainerRuntimeEndpoint(containerRuntime string) []string {
	switch containerRuntime {
	case "docker":
		return []string{
			"unix:///var/run/dockershim.sock",
			"unix:///var/run/cri-dockerd.sock",
			"unix:///var/run/docker.sock",
		}
	case "containerd":
		return []string{"unix:///run/containerd/containerd.sock"}
	case "crio":
		return []string{"unix:///run/crio/crio.sock"}
	}

	return nil
}

// NewRemoteRuntimeService only support unix socket endpoint
func NewRemoteRuntimeService(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, endpoint,
		grpc.WithContextDialer(unixConnect),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func unixConnect(ctx context.Context, sockEndpoint string) (net.Conn, error) {
	protocol, addr, err := parseEndpoint(sockEndpoint)
	if err != nil {
		return nil, err
	}
	if protocol != unixProtocol {
		return nil, fmt.Errorf("only support unix socket endpoint")
	}
	unixAddress, err := net.ResolveUnixAddr(protocol, addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUnix(protocol, nil, unixAddress)
	return conn, err
}

func parseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", err
	}

	switch u.Scheme {
	case "tcp":
		return "tcp", u.Host, nil

	case "unix":
		return "unix", u.Path, nil

	case "":
		return "", "", fmt.Errorf("using %q as endpoint is deprecated, please consider using full url format", endpoint)

	default:
		return u.Scheme, "", fmt.Errorf("protocol %q not supported", u.Scheme)
	}
}

func Info(conn *grpc.ClientConn) (*runtimeapi.StatusResponse, error) {
	out := new(runtimeapi.StatusResponse)
	err := conn.Invoke(context.TODO(), "/runtime.v1.RuntimeService/Status", &runtimeapi.StatusRequest{
		Verbose: true}, out)
	return out, err
}
