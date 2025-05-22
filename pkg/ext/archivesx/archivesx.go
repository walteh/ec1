package archivesx

import (
	"context"
	"io"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ext/iox"
)

type Manipulator interface {
	Manipulate(ctx context.Context, reader io.Reader) (io.ReadCloser, error)
}

func IdentifyAndDecompress(ctx context.Context, path string, reader io.Reader) (io.ReadCloser, bool, error) {
	format, rdr, err := archives.Identify(ctx, path, reader)
	if err != nil {
		return nil, false, errors.Errorf("identifying file %s: %w", path, err)
	}

	if err == archives.NoMatch {
		return iox.PreservedNopCloser(rdr), false, nil
	}

	if format, ok := format.(archives.Compression); ok {
		rdrz, err := format.OpenReader(rdr)
		if err != nil {
			return nil, false, errors.Errorf("opening compression reader: %w", err)
		}
		return rdrz, true, nil
	}

	return nil, false, errors.Errorf("unable to decompress format %T", format)
}

// CreateCompressorPipeline creates a pipeline that compresses data from the given reader.
// The returned ReadCloser provides access to the compressed data.
func CreateCompressorPipeline(ctx context.Context, compression archives.Compression, reader io.Reader) (io.ReadCloser, error) {
	return iox.CreateWriterPipeline(ctx, reader, func(writer io.Writer) (io.WriteCloser, error) {
		return compression.OpenWriter(writer)
	})
}
