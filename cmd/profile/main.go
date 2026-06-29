package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/achmudas/identity-api/gen/profile/v1"
	"github.com/achmudas/identity-api/gen/profile/v1/profilev1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type profileServiceServer struct {
	profilev1connect.UnimplementedProfileServiceHandler
}

func (ps *profileServiceServer) GetProfileData(ctx context.Context, req *v1.GetProfileDataRequest) (*v1.GetProfileDataResponse, error) {
	// #TODO would make sense to create actual service, sqls, etc. but too lazy. Where I should store them?
	return &v1.GetProfileDataResponse{Profile: &v1.Profile{UserId: req.UserId, AvatarLink: "http://localhost:8085/me/1.png", Address: "address1, city1", BirthDate: timestamppb.New(time.Date(
		2009, 11, 17, 20, 34, 58, 651387237, time.UTC))}}, nil
}

func loggingInterceptor() connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (n connect.AnyResponse, err error) {
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				requestIDs := md.Get("x-request-id")
				if len(requestIDs) > 0 {
					log.Printf("request-id: %s procedure: %s", requestIDs[0], req.Spec().Procedure)
				}
			}
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic: %v\n%s", r, debug.Stack())
					err = connect.NewError(connect.CodeInternal, fmt.Errorf("panic: %v", r))
				}
			}()
			return next(ctx, req)
		})
	})
}

const address = "localhost:8085"

func main() {
	mux := http.NewServeMux()
	path, handler := profilev1connect.NewProfileServiceHandler(&profileServiceServer{}, connect.WithInterceptors(loggingInterceptor()))
	mux.Handle(path, handler)
	srv := &http.Server{Addr: address, Handler: h2c.NewHandler(mux, &http2.Server{})}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")

}
