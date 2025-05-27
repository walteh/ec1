package oci

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

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/containerd/containerd/v2/core/diff"
	"github.com/containerd/containerd/v2/core/leases"
	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/errdefs"
	"github.com/containerd/platforms"
	"github.com/opencontainers/image-spec/identity"

	containerd "github.com/containerd/containerd/v2/client"
)

type RunnerOpts struct {
	Target      string
	Ref         string
	Platform    platforms.Platform
	Rw          bool
	Snapshotter string
}

func Runner(ctx context.Context, opts RunnerOpts) (retErr error) {

	if opts.Ref == "" {
		return errors.New("please provide an image reference to mount")
	}
	if opts.Target == "" {
		return errors.New("please provide a target path to mount to")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, err := containerd.New("/var/run/docker.sock")
	if err != nil {
		return err
	}
	defer cancel()

	ctx, done, err := client.WithLease(ctx,
		leases.WithID(opts.Target),
		leases.WithExpiration(24*time.Hour),
		leases.WithLabel("containerd.io/gc.ref.snapshot."+opts.Snapshotter, opts.Target),
	)
	if err != nil && !errdefs.IsAlreadyExists(err) {
		return err
	}

	defer func() {
		if retErr != nil && done != nil {
			done(ctx)
		}
	}()

	img, err := client.ImageService().Get(ctx, opts.Ref)
	if err != nil {
		return err
	}

	i := containerd.NewImageWithPlatform(client, img, platforms.Only(opts.Platform))
	if err := i.Unpack(ctx, opts.Snapshotter, containerd.WithUnpackApplyOpts(diff.WithSyncFs(false))); err != nil {
		return fmt.Errorf("error unpacking image: %w", err)
	}

	diffIDs, err := i.RootFS(ctx)
	if err != nil {
		return err
	}
	chainID := identity.ChainID(diffIDs).String()
	fmt.Println(chainID)

	s := client.SnapshotService(opts.Snapshotter)

	var mounts []mount.Mount
	if opts.Rw {
		mounts, err = s.Prepare(ctx, opts.Target, chainID)
	} else {
		mounts, err = s.View(ctx, opts.Target, chainID)
	}
	if err != nil {
		if errdefs.IsAlreadyExists(err) {
			mounts, err = s.Mounts(ctx, opts.Target)
		}
		if err != nil {
			return err
		}
	}

	if err := mount.All(mounts, opts.Target); err != nil {
		if err := s.Remove(ctx, opts.Target); err != nil && !errdefs.IsNotFound(err) {
			fmt.Fprintln(os.Stderr, "Error cleaning up snapshot after mount error:", err)
		}
		return err
	}

	// fmt.Fprintln(cliContext.App.Writer, opts.Target)
	return nil
}
