package containerd

import (
	"context"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	cruntime "github.com/containerd/containerd/v2/core/runtime/v2"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/ttrpc"
	slogctx "github.com/veqryn/slog-context"
	"gitlab.com/tozd/go/errors"

	"github.com/containerd/containerd/api/runtime/task/v3"
	"google.golang.org/protobuf/types/known/emptypb"
)

type taskService interface {
	task.TTRPCTaskService
	cruntime.TaskServiceClient
	shim.TTRPCService
}

var _ task.TTRPCTaskService = &errTaskService{}
var _ cruntime.TaskServiceClient = &errTaskService{}
var _ shim.TTRPCService = &errTaskService{}

type errTaskService struct {
	ref              taskService
	enableLogErrors  bool
	enableLogSuccess bool
}

// RegisterTTRPC implements shim.TTRPCService.
func (e *errTaskService) RegisterTTRPC(s *ttrpc.Server) error {
	s.SetDebugging(true)
	task.RegisterTTRPCTaskService(s, e)
	return nil
}

func WrapTaskServiceWithErrorLogging(s taskService) taskService {
	return &errTaskService{
		ref:              s,
		enableLogErrors:  true,
		enableLogSuccess: true,
	}
}

func wrap[I, O any](e *errTaskService, f func(context.Context, I) (O, error)) func(context.Context, I) (O, error) {

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

// Checkpoint implements task.TTRPCTaskService.
func (e *errTaskService) Checkpoint(ctx context.Context, req *task.CheckpointTaskRequest) (*emptypb.Empty, error) {
	return wrap(e, e.ref.Checkpoint)(ctx, req)
}

// CloseIO implements task.TTRPCTaskService.
func (e *errTaskService) CloseIO(ctx context.Context, req *task.CloseIORequest) (*emptypb.Empty, error) {
	return wrap(e, e.ref.CloseIO)(ctx, req)
}

// Connect implements task.TTRPCTaskService.
func (e *errTaskService) Connect(ctx context.Context, req *task.ConnectRequest) (*task.ConnectResponse, error) {
	return wrap(e, e.ref.Connect)(ctx, req)
}

// Create implements task.TTRPCTaskService.
func (e *errTaskService) Create(ctx context.Context, req *task.CreateTaskRequest) (*task.CreateTaskResponse, error) {
	return wrap(e, e.ref.Create)(ctx, req)
}

// Delete implements task.TTRPCTaskService.
func (e *errTaskService) Delete(ctx context.Context, req *task.DeleteRequest) (*task.DeleteResponse, error) {
	return wrap(e, e.ref.Delete)(ctx, req)
}

// Exec implements task.TTRPCTaskService.
func (e *errTaskService) Exec(ctx context.Context, req *task.ExecProcessRequest) (*emptypb.Empty, error) {
	return wrap(e, e.ref.Exec)(ctx, req)
}

// Kill implements task.TTRPCTaskService.
func (e *errTaskService) Kill(ctx context.Context, req *task.KillRequest) (*emptypb.Empty, error) {
	return wrap(e, e.ref.Kill)(ctx, req)
}

// Pause implements task.TTRPCTaskService.
func (e *errTaskService) Pause(ctx context.Context, req *task.PauseRequest) (*emptypb.Empty, error) {
	return wrap(e, e.ref.Pause)(ctx, req)
}

// Pids implements task.TTRPCTaskService.
func (e *errTaskService) Pids(ctx context.Context, req *task.PidsRequest) (*task.PidsResponse, error) {
	return wrap(e, e.ref.Pids)(ctx, req)
}

// ResizePty implements task.TTRPCTaskService.
func (e *errTaskService) ResizePty(ctx context.Context, req *task.ResizePtyRequest) (*emptypb.Empty, error) {
	return wrap(e, e.ref.ResizePty)(ctx, req)
}

// Resume implements task.TTRPCTaskService.
func (e *errTaskService) Resume(ctx context.Context, req *task.ResumeRequest) (*emptypb.Empty, error) {
	return wrap(e, e.ref.Resume)(ctx, req)
}

// Shutdown implements task.TTRPCTaskService.
func (e *errTaskService) Shutdown(ctx context.Context, req *task.ShutdownRequest) (*emptypb.Empty, error) {
	return wrap(e, e.ref.Shutdown)(ctx, req)
}

// Start implements task.TTRPCTaskService.
func (e *errTaskService) Start(ctx context.Context, req *task.StartRequest) (*task.StartResponse, error) {
	return wrap(e, e.ref.Start)(ctx, req)
}

// State implements task.TTRPCTaskService.
func (e *errTaskService) State(ctx context.Context, req *task.StateRequest) (*task.StateResponse, error) {
	return wrap(e, e.ref.State)(ctx, req)
}

// Stats implements task.TTRPCTaskService.
func (e *errTaskService) Stats(ctx context.Context, req *task.StatsRequest) (*task.StatsResponse, error) {
	return wrap(e, e.ref.Stats)(ctx, req)
}

// Update implements task.TTRPCTaskService.
func (e *errTaskService) Update(ctx context.Context, req *task.UpdateTaskRequest) (*emptypb.Empty, error) {
	return wrap(e, e.ref.Update)(ctx, req)
}

// Wait implements task.TTRPCTaskService.
func (e *errTaskService) Wait(ctx context.Context, req *task.WaitRequest) (*task.WaitResponse, error) {
	return wrap(e, e.ref.Wait)(ctx, req)
}
