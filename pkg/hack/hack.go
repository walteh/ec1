package hack

import (
	"log/slog"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/lmittmann/tint"
)

func GetUnexportedFieldOf(obj any, field string) any {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return GetUnexportedField(val.FieldByName(field))
}

func GetUnexportedField(field reflect.Value) any {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func SetUnexportedField(field reflect.Value, value any) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).
		Elem().
		Set(reflect.ValueOf(value))
}

func getFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

func MyCaller() runtime.Frame {
	// Skip GetCallerFunctionName and the function to get the caller of
	return getFrame(2)
}

func GetPrevFunctionCaller() runtime.Frame {
	// prepare a slice to hold return PCs
	pcs := make([]uintptr, 10)
	// skip: 0=runtime.Callers, 1=this wrapAndLocate frame,
	// so 2 is the caller *inside* f, where it returned
	// 3 is the caller of the caller of the caller
	n := runtime.Callers(0, pcs)
	if n == 0 {
		return runtime.Frame{Function: "unknown"}
	}

	more := true
	var frame runtime.Frame
	frames := runtime.CallersFrames(pcs[:n])

	// take the first frame
	for more {
		frame, more = frames.Next()
		slog.Info("GetPrevFunctionCaller", "frame", tint.NewPrettyValue(frame))

		if !more {
			break
		}
	}
	return frame
}
