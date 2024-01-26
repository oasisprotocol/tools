package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	oasisGrpc "github.com/oasisprotocol/oasis-core/go/common/grpc"
	consensus "github.com/oasisprotocol/oasis-core/go/consensus/api"
)

func exitErr(err error) {
	if err != nil {
		fmt.Println("err:", err)
		debug.PrintStack()
		os.Exit(1)
	}
}

type basicAuth struct {
	username string
	password string
}

func (b basicAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	auth := b.username + ":" + b.password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))
	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

func (basicAuth) RequireTransportSecurity() bool {
	return true
}

func getGrpcDialOpts(grpcTlsMode string, grpcTlsCustomCa string, grpcUsername string, grpcPassword string, grpcKeepaliveTime int) []grpc.DialOption {
	grpcDialOpts := []grpc.DialOption{}

	switch grpcTlsMode {
	case "off": // Use non-encrypted connection
		grpcDialOpts = append(grpcDialOpts, grpc.WithInsecure())

	case "off-alt": // Use non-encrypted connection (alternative)
		grpcDialOpts = append(grpcDialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	case "insecure": // Use TLS without certificate checks
		grpcDialOpts = append(grpcDialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))

	case "ca": // Use TLS with custom CA certificate file
		certCreds, _ := credentials.NewClientTLSFromFile(grpcTlsCustomCa, "")
		grpcDialOpts = append(grpcDialOpts, grpc.WithTransportCredentials(certCreds))

	case "", "system": // Use TLS with system certificates and TLS v1.2
		grpcDialOpts = append(grpcDialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})))

	case "system-alt": // Use TLS with system certificates (alternative)
		certPool, _ := x509.SystemCertPool()
		grpcDialOpts = append(grpcDialOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(certPool, "")))
	}

	if grpcUsername != "" {
		// Use HTTP Basic authentication
		grpcDialOpts = append(grpcDialOpts, grpc.WithPerRPCCredentials(basicAuth{
			username: grpcUsername,
			password: grpcPassword,
		}))
	}

	if grpcKeepaliveTime > 0 {
		// Send keep-alive requests
		grpcDialOpts = append(grpcDialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Duration(grpcKeepaliveTime) * time.Second,
			Timeout:             time.Duration(grpcKeepaliveTime) * time.Second,
			PermitWithoutStream: true,
		}))
	}
	return grpcDialOpts
}

func main() {
	var err error

	// Parse options
	grpcAddress := "unix:internal.sock"
	grpcTlsMode := "off"
	grpcTlsCustomCa := "ca.crt"
	grpcUsername := ""
	grpcPassword := ""
	grpcKeepaliveTime := 0

	if len(os.Args) < 3 {
		fmt.Println("usage: ./grpc-tls-client <grpc_address> <tls_mode> [<http_username> <http_password> [<keepalive_time>]]")
		os.Exit(2)
	} else {
		grpcAddress = os.Args[1]
		grpcTlsMode = os.Args[2]
	}
	if len(os.Args) >= 5 {
		grpcUsername = os.Args[3]
		grpcPassword = os.Args[4]
	}
	if len(os.Args) >= 6 {
		grpcKeepaliveTime, err = strconv.Atoi(os.Args[5])
		exitErr(err)
	}

	// Connect to gRPC endpoint
	ctx := context.Background()
	fmt.Println("connect to", grpcAddress)
	grpcDialOpts := getGrpcDialOpts(grpcTlsMode, grpcTlsCustomCa, grpcUsername, grpcPassword, grpcKeepaliveTime)
	nodeConn, err := oasisGrpc.Dial(grpcAddress, grpcDialOpts...)
	exitErr(err)
	cons := consensus.NewConsensusClient(nodeConn)

	// Get chain context
	chainContext, err := cons.GetChainContext(ctx)
	exitErr(err)
	fmt.Println("chain context", chainContext)

	// Get node status (genesis height and latest height)
	status, err := cons.GetStatus(ctx)
	exitErr(err)
	fmt.Println("latest height", status.LatestHeight)
	fmt.Println("node status", status)
}
