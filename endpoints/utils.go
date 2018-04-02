package endpoints

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc/peer"
)

func fixPrefix(s string) string {
	l := len(s)
	if l == 0 {
		return s
	}
	if s[l-1] == '/' {
		return s
	}
	return s + "/"
}

func getRemoteAddr(ctx context.Context) (string, error) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("failed to get peer from context")
	}
	if pr.Addr == net.Addr(nil) {
		return "", fmt.Errorf("failed to get peer address")
	}
	return pr.Addr.String(), nil
}
