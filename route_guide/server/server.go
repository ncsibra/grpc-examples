package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	_ "embed"

	"google.golang.org/grpc"

	pb "github.com/ncsibra/grpc-examples/route_guide/routeguide"
)

var (
	port = flag.Int("port", 50051, "The server port")

	//go:embed testdata/route_guide_db.json
	exampleData []byte
)

type routeGuideServer struct {
	pb.UnimplementedRouteGuideServer
	savedFeatures []*pb.Feature // read-only after initialized
}

func (s *routeGuideServer) ListFeatures(_ *pb.ListFeaturesRequest, stream pb.RouteGuide_ListFeaturesServer) error {
	ctx := stream.Context()

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i, feature := range s.savedFeatures {
			if err := stream.Send(feature); err != nil {
				if errors.Is(err, context.Canceled) {
					log.Println("context canceled during send")
				} else {
					log.Printf("sending error: %v\n", err)
				}
				return
			}

			log.Printf("%d. feature sent\n", i)

			select {
			case <-ctx.Done():
				log.Println("context canceled during sleep")
				return
			case <-time.After(100 * time.Millisecond):
				// continue
			}
		}
	}()

	wg.Wait()

	log.Println("ListFeatures finished")

	return nil
}

// loadFeatures loads features from a JSON file.
func (s *routeGuideServer) loadFeatures() {
	if err := json.Unmarshal(exampleData, &s.savedFeatures); err != nil {
		log.Fatalf("Failed to load default features: %v", err)
	}
}

func newServer() *routeGuideServer {
	s := &routeGuideServer{}
	s.loadFeatures()
	return s
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRouteGuideServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}
