package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a *Agent) FileTransfer(ctx context.Context, stream *connect.ClientStream[v1poc1.FileTransferRequest]) (*connect.Response[v1poc1.FileTransferResponse], error) {
	req := stream.Msg()

	// if the file already exists, return success
	if _, err := os.Stat(filepath.Join("/tmp", req.GetName())); err == nil {
		return connect.NewResponse(&v1poc1.FileTransferResponse{Exists: ptr(true)}), nil
	}

	openFile, err := os.Open(filepath.Join("/tmp", req.GetName()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer openFile.Close()

	hash := sha256.New()

	multiWriter := io.MultiWriter(openFile, hash)

	fileSize := req.GetFileByteSize()
	if _, err := openFile.Write(req.GetChunkBytes()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for stream.Receive() {
		fileSize += stream.Msg().GetChunkByteSize()
		if _, err := multiWriter.Write(stream.Msg().GetChunkBytes()); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	fileName := filepath.Base(req.GetName())
	fmt.Printf("saved file: %s, size: %d\n", fileName, fileSize)
	return connect.NewResponse(&v1poc1.FileTransferResponse{Exists: ptr(true), Size: ptr(fileSize), Hash: ptr(hex.EncodeToString(hash.Sum(nil)))}), nil
}
