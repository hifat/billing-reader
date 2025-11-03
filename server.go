package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"path/filepath"

	"github.com/hifat/billing-reader/pb"
	"github.com/hifat/billing-reader/utils"
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

func (s *server) ReadReceipt(ctx context.Context, stream pb.BillingReader_ReadReceiptServer) (*pb.ReadReceiptResponse, error) {
	// if len(req.Chunk) > MaxFileSize {
	// 	return &pb.ReadReceiptResponse{
	// 		Success: false,
	// 		Error:   fmt.Sprintf("file size exceeds maximum of %d bytes", MaxFileSize),
	// 	}, nil
	// }

	file := utils.File{
		FilePath:   "",
		OutputFile: nil,
	}

	defer func() {
		if err := file.OutputFile.Close(); err != nil {
			log.Printf("Failed to close output file: %v", err)
		}
	}()

	var fileSize uint32 = 0
	for {
		req, err := stream.Recv()
		if file.FilePath == "" {
			file.SetFile(req.GetFileName(), "./upload")
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return &pb.ReadReceiptResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to receive file: %v", err),
			}, nil
		}

		chunk := req.GetChunk()
		fileSize += uint32(len(chunk))
		slog.Debug("received a chunk with size: ", "bytes", fileSize)
		if err := file.Write(chunk); err != nil {
			return &pb.ReadReceiptResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to write file: %v", err),
			}, nil
		}
	}

	fileName := filepath.Base(file.FilePath)
	slog.Debug("saved file: %s, size: %d", fileName, fileSize)

	err := stream.SendAndClose(&pb.ReadReceiptResponse{
		Success: true,
		Error:   "",
	})

	_ = err

	/* ----------------------------- End Upload Part ---------------------------- */

	// receipts, err := ExtractFromImage(ctx, "./env/cloud-vision-srv-scc.json", "./upload/receipt.jpg")
	// if err != nil {
	// 	log.Printf("Failed to extract from image: %v", err)
	// 	return &pb.ReadReceiptResponse{
	// 		Success: false,
	// 		Error:   fmt.Sprintf("failed to extract from image: %v", err),
	// 	}, nil
	// }

	// resReceipts := make([]*pb.Receipt, 0, len(receipts))
	// for _, v := range receipts {
	// 	resReceipts = append(resReceipts, &pb.Receipt{
	// 		Name:             v.Name,
	// 		NameEdited:       v.NameEdited,
	// 		PurchasePrice:    v.PurchasePrice,
	// 		PurchaseQuantity: v.PurchaseQuantity,
	// 		Remark:           v.Remark,
	// 	})
	// }

	return &pb.ReadReceiptResponse{
		Success: true,
		Receipt: nil,
	}, nil
}
