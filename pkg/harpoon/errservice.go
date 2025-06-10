package harpoon

import (
	"context"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	slogctx "github.com/veqryn/slog-context"
	"gitlab.com/tozd/go/errors"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
)

var (
	_ harpoonv1.TTRPCGuestServiceService = &errService{}
)

type errService struct {
	ref              harpoonv1.TTRPCGuestServiceService
	enableLogErrors  bool
	enableLogSuccess bool
}

// Exec implements harpoonv1.TTRPCGuestServiceService.
func (e *errService) Exec(ctx context.Context, server harpoonv1.TTRPCGuestService_ExecServer) error {
	return streamWrap(e, e.ref.Exec)(ctx, server)
}

// Readiness implements harpoonv1.TTRPCGuestServiceService.
func (e *errService) Readiness(ctx context.Context, req *harpoonv1.ReadinessRequest) (*harpoonv1.ReadinessResponse, error) {
	return wrap(e, e.ref.Readiness)(ctx, req)
}

// Run implements harpoonv1.TTRPCGuestServiceService.
func (e *errService) Run(ctx context.Context, req *harpoonv1.RunRequest) (*harpoonv1.RunResponse, error) {
	return wrap(e, e.ref.Run)(ctx, req)
}

// TimeSync implements harpoonv1.TTRPCGuestServiceService.
func (e *errService) TimeSync(ctx context.Context, req *harpoonv1.TimeSyncRequest) (*harpoonv1.TimeSyncResponse, error) {
	return wrap(e, e.ref.TimeSync)(ctx, req)
}

func WrapGuestServiceWithErrorLogging(s harpoonv1.TTRPCGuestServiceService) harpoonv1.TTRPCGuestServiceService {
	return &errService{
		ref:              s,
		enableLogErrors:  true,
		enableLogSuccess: true,
	}
}

func streamWrap[I any](e *errService, f func(context.Context, I) error) func(context.Context, I) error {
	pc, _, _, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	realNameS := strings.Split(filepath.Base(funcName), ".")
	realName := realNameS[len(realNameS)-1]
	return func(ctx context.Context, req I) error {
		start := time.Now()

		ctx = slogctx.Append(ctx, slog.String("ttrpc_method", realName))

		err := f(ctx, req)

		end := time.Now()

		if err != nil && e.enableLogErrors {
			if trac, ok := err.(errors.E); ok {
				pc = trac.StackTrace()[0]
			}

			rec := slog.NewRecord(end, slog.LevelError, "error in task service", pc)
			rec.AddAttrs(
				slog.String("error", err.Error()),
				slog.String("method", realName),
				slog.Duration("duration", end.Sub(start)),
			)
			if err := slog.Default().Handler().Handle(ctx, rec); err != nil {
				slog.ErrorContext(ctx, "error logging error", "error", err)
			}
		}
		if err == nil && e.enableLogSuccess {
			rec := slog.NewRecord(end, slog.LevelInfo, "success in task service", pc)
			rec.AddAttrs(
				slog.String("method", realName),
				slog.Duration("duration", end.Sub(start)),
			)
			if err := slog.Default().Handler().Handle(ctx, rec); err != nil {
				slog.ErrorContext(ctx, "error logging success", "error", err)
			}
		}

		return err
	}
}

func wrap[I, O any](e *errService, f func(context.Context, I) (O, error)) func(context.Context, I) (O, error) {

	pc, _, _, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	realNameS := strings.Split(filepath.Base(funcName), ".")
	realName := realNameS[len(realNameS)-1]

	return func(ctx context.Context, req I) (O, error) {
		start := time.Now()

		ctx = slogctx.Append(ctx, slog.String("ttrpc_method", realName))

		resp, err := f(ctx, req)

		end := time.Now()

		if err != nil && e.enableLogErrors {
			if trac, ok := err.(errors.E); ok {
				pc = trac.StackTrace()[0]
			}

			rec := slog.NewRecord(end, slog.LevelError, "error in task service", pc)
			rec.AddAttrs(
				slog.String("error", err.Error()),
				slog.String("method", realName),
				slog.Duration("duration", end.Sub(start)),
			)
			if err := slog.Default().Handler().Handle(ctx, rec); err != nil {
				slog.ErrorContext(ctx, "error logging error", "error", err)
			}
		}
		if err == nil && e.enableLogSuccess {
			rec := slog.NewRecord(end, slog.LevelInfo, "success in task service", pc)
			rec.AddAttrs(
				slog.String("method", realName),
				slog.Duration("duration", end.Sub(start)),
			)
			if err := slog.Default().Handler().Handle(ctx, rec); err != nil {
				slog.ErrorContext(ctx, "error logging success", "error", err)
			}
		}
		return resp, nil
	}
}
