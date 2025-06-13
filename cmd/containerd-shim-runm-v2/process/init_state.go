//go:build !windows

/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package process

import (
	"context"
	"errors"
	"fmt"

	google_protobuf "github.com/containerd/containerd/v2/pkg/protobuf/types"
)

type initState interface {
	Start(context.Context) error
	Delete(context.Context) error
	Pause(context.Context) error
	Resume(context.Context) error
	Update(context.Context, *google_protobuf.Any) error
	Checkpoint(context.Context, *CheckpointConfig) error
	Exec(context.Context, string, *ExecConfig) (Process, error)
	Kill(context.Context, uint32, bool) error
	SetExited(int)
	Status(context.Context) (string, error)
}

type createdState struct {
	p *Init
}

func (s *createdState) transition(name string) error {
	switch name {
	case "running":
		s.p.initState = &runningState{p: s.p}
	case "stopped":
		s.p.initState = &stoppedState{p: s.p}
	case "deleted":
		s.p.initState = &deletedState{}
	default:
		return fmt.Errorf("invalid state transition %q to %q", stateName(s), name)
	}
	return nil
}

func (s *createdState) Pause(ctx context.Context) error {
	return errors.New("cannot pause task in created state")
}

func (s *createdState) Resume(ctx context.Context) error {
	return errors.New("cannot resume task in created state")
}

func (s *createdState) Update(ctx context.Context, r *google_protobuf.Any) error {
	return s.p.update(ctx, r)
}

func (s *createdState) Checkpoint(ctx context.Context, r *CheckpointConfig) error {
	return errors.New("cannot checkpoint a task in created state")
}

func (s *createdState) Start(ctx context.Context) error {
	if err := s.p.start(ctx); err != nil {
		return err
	}
	return s.transition("running")
}

func (s *createdState) Delete(ctx context.Context) error {
	if err := s.p.delete(ctx); err != nil {
		return err
	}
	return s.transition("deleted")
}

func (s *createdState) Kill(ctx context.Context, sig uint32, all bool) error {
	return s.p.kill(ctx, sig, all)
}

func (s *createdState) SetExited(status int) {
	s.p.setExited(status)

	if err := s.transition("stopped"); err != nil {
		panic(err)
	}
}

func (s *createdState) Exec(ctx context.Context, path string, r *ExecConfig) (Process, error) {
	return s.p.exec(ctx, path, r)
}

func (s *createdState) Status(ctx context.Context) (string, error) {
	return "created", nil
}

type runningState struct {
	p *Init
}

func (s *runningState) transition(name string) error {
	switch name {
	case "stopped":
		s.p.initState = &stoppedState{p: s.p}
	case "paused":
		s.p.initState = &pausedState{p: s.p}
	default:
		return fmt.Errorf("invalid state transition %q to %q", stateName(s), name)
	}
	return nil
}

func (s *runningState) Pause(ctx context.Context) error {
	s.p.pausing.Store(true)
	// NOTE "pausing" will be returned in the short window
	// after `transition("paused")`, before `pausing` is reset
	// to false. That doesn't break the state machine, just
	// delays the "paused" state a little bit.
	defer s.p.pausing.Store(false)

	// For VMs, pause would be handled differently
	if s.p.vm != nil {
		// This would call our VM pause logic
		// For now, this is a placeholder
	}

	return s.transition("paused")
}

func (s *runningState) Resume(ctx context.Context) error {
	return errors.New("cannot resume a running process")
}

func (s *runningState) Update(ctx context.Context, r *google_protobuf.Any) error {
	return s.p.update(ctx, r)
}

func (s *runningState) Checkpoint(ctx context.Context, r *CheckpointConfig) error {
	return s.p.checkpoint(ctx, r)
}

func (s *runningState) Start(ctx context.Context) error {
	return errors.New("cannot start a running process")
}

func (s *runningState) Delete(ctx context.Context) error {
	return errors.New("cannot delete a running process")
}

func (s *runningState) Kill(ctx context.Context, sig uint32, all bool) error {
	return s.p.kill(ctx, sig, all)
}

func (s *runningState) SetExited(status int) {
	s.p.setExited(status)

	if err := s.transition("stopped"); err != nil {
		panic(err)
	}
}

func (s *runningState) Exec(ctx context.Context, path string, r *ExecConfig) (Process, error) {
	return s.p.exec(ctx, path, r)
}

func (s *runningState) Status(ctx context.Context) (string, error) {
	return "running", nil
}

type pausedState struct {
	p *Init
}

func (s *pausedState) transition(name string) error {
	switch name {
	case "running":
		s.p.initState = &runningState{p: s.p}
	case "stopped":
		s.p.initState = &stoppedState{p: s.p}
	default:
		return fmt.Errorf("invalid state transition %q to %q", stateName(s), name)
	}
	return nil
}

func (s *pausedState) Pause(ctx context.Context) error {
	return errors.New("cannot pause a paused container")
}

func (s *pausedState) Resume(ctx context.Context) error {
	// For VMs, resume would be handled differently
	if s.p.vm != nil {
		// This would call our VM resume logic
		// For now, this is a placeholder
	}

	return s.transition("running")
}

func (s *pausedState) Update(ctx context.Context, r *google_protobuf.Any) error {
	return s.p.update(ctx, r)
}

func (s *pausedState) Checkpoint(ctx context.Context, r *CheckpointConfig) error {
	return s.p.checkpoint(ctx, r)
}

func (s *pausedState) Start(ctx context.Context) error {
	return errors.New("cannot start a paused process")
}

func (s *pausedState) Delete(ctx context.Context) error {
	return errors.New("cannot delete a paused process")
}

func (s *pausedState) Kill(ctx context.Context, sig uint32, all bool) error {
	return s.p.kill(ctx, sig, all)
}

func (s *pausedState) SetExited(status int) {
	s.p.setExited(status)

	// For VMs, we might need to resume before transitioning to stopped
	if s.p.vm != nil {
		// This would call our VM resume logic if needed
		// For now, this is a placeholder
	}

	if err := s.transition("stopped"); err != nil {
		panic(err)
	}
}

func (s *pausedState) Exec(ctx context.Context, path string, r *ExecConfig) (Process, error) {
	return nil, errors.New("cannot exec in a paused state")
}

func (s *pausedState) Status(ctx context.Context) (string, error) {
	return "paused", nil
}

type stoppedState struct {
	p *Init
}

func (s *stoppedState) transition(name string) error {
	switch name {
	case "deleted":
		s.p.initState = &deletedState{}
	default:
		return fmt.Errorf("invalid state transition %q to %q", stateName(s), name)
	}
	return nil
}

func (s *stoppedState) Pause(ctx context.Context) error {
	return errors.New("cannot pause a stopped container")
}

func (s *stoppedState) Resume(ctx context.Context) error {
	return errors.New("cannot resume a stopped container")
}

func (s *stoppedState) Update(ctx context.Context, r *google_protobuf.Any) error {
	return errors.New("cannot update a stopped container")
}

func (s *stoppedState) Checkpoint(ctx context.Context, r *CheckpointConfig) error {
	return errors.New("cannot checkpoint a stopped container")
}

func (s *stoppedState) Start(ctx context.Context) error {
	return errors.New("cannot start a stopped process")
}

func (s *stoppedState) Delete(ctx context.Context) error {
	if err := s.p.delete(ctx); err != nil {
		return err
	}
	return s.transition("deleted")
}

func (s *stoppedState) Kill(ctx context.Context, sig uint32, all bool) error {
	return s.p.kill(ctx, sig, all)
}

func (s *stoppedState) SetExited(status int) {
	// no op
}

func (s *stoppedState) Exec(ctx context.Context, path string, r *ExecConfig) (Process, error) {
	return nil, errors.New("cannot exec in a stopped state")
}

func (s *stoppedState) Status(ctx context.Context) (string, error) {
	return "stopped", nil
}
