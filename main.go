package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/hifat/billing-reader/pb"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

type OCRResult struct {
	FullText string      `json:"text"`
	Blocks   []TextBlock `json:"blocks"`
}

type TextBlock struct {
	Text        string  `json:"text"`
	Confidence  float32 `json:"confidence"`
	BoundingBox []Point `json:"bounding_box"`
}

type Point struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

func init() {
	if err := godotenv.Load("./env/.env"); err != nil {
		log.Println("Running without .env file (using system environment variables)")
	}
}

func main() {
	s := &server{
		apiKey: os.Getenv("API_KEY"),
	}

	port := os.Getenv("PORT")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(s.authInterceptor),
	)

	pb.RegisterBillingReaderServer(grpcSrv, &server{})

	log.Printf("listening on port :%s", port)
	if err := grpcSrv.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
