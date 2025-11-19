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

func (s *server) ReadReceipt(stream pb.BillingReader_ReadReceiptServer) error {
	// TODO: Validate file size

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
			log.Printf("Failed to stream receive: %v", err)
			status.Error(codes.Internal, fmt.Sprintf("failed to receive file: %v", err))
		}

		chunk := req.GetChunk()
		fileSize += uint32(len(chunk))
		slog.Debug("received a chunk with size: ", "bytes", fileSize)
		if err := file.Write(chunk); err != nil {
			log.Printf("Failed to write chunk: %v", err)
			status.Error(codes.Internal, fmt.Sprintf("Failed to write chunk: %v", err))
		}
	}
	fileName := filepath.Base(file.FilePath)
	slog.Debug("saved file: %s, size: %d", fileName, fileSize)

	/* ----------------------------- End Upload Part ---------------------------- */

	receipts, err := ExtractFromImage(stream.Context(), "./env/cloud-vision-srv-scc.json", "./"+file.FilePath)
	if err != nil {
		log.Printf("Failed to extract from image: %v", err)
		status.Error(codes.Internal, fmt.Sprintf("failed to extract from image: %v", err))
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

	err = stream.SendAndClose(&pb.ReadReceiptResponse{
		Success: true,
		Receipt: resReceipts,
	})
	if err != nil {
		log.Printf("Failed to SendAndClose: %v", err)
		status.Error(codes.Internal, fmt.Sprintf("Failed to SendAndClose: %v", err))
	}

	// TODO: Remove file when used

	return nil
}
