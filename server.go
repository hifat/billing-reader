package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hifat/billing-reader/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const MaxFileSize = 10 * 1024 * 1024 // 10 MB

type server struct {
	pb.BillingReaderServer
	apiKey string
}

func (s *server) authInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "messing metadata")
	}

	apiKeys := md.Get("x-api-key")
	if len(apiKeys) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing api key")
	}

	if apiKeys[0] != s.apiKey {
		return nil, status.Error(codes.Unauthenticated, "invalid api key")
	}

	return handler(ctx, req)
}

func (s *server) ReadReceipt(ctx context.Context, req *pb.ReadReceiptRequest) (*pb.ReadReceiptResponse, error) {
	if len(req.ImageData) > MaxFileSize {
		return &pb.ReadReceiptResponse{
			Success: false,
			Error:   fmt.Sprintf("file size exceeds maximum of %d bytes", MaxFileSize),
		}, nil
	}

	receipts, err := ExtractFromImage(ctx, "./env/cloud-vision-srv-scc.json", "./upload/receipt.jpg")
	if err != nil {
		log.Printf("Failed to extract from image: %v", err)
		return &pb.ReadReceiptResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to extract from image: %v", err),
		}, nil
	}

	resReceipts := make([]*pb.Receipt, 0, len(receipts))
	for _, v := range receipts {
		resReceipts = append(resReceipts, &pb.Receipt{
			Name:             v.Name,
			NameEdited:       v.NameEdited,
			PurchasePrice:    v.PurchasePrice,
			PurchaseQuantity: v.PurchaseQuantity,
			Remark:           v.Remark,
		})
	}

	return &pb.ReadReceiptResponse{
		Success: true,
		Receipt: resReceipts,
	}, nil
}
