package zapsentry

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pkg/errors"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Test_extractTrace(t *testing.T) {
	type args struct {
		fs []zapcore.Field
	}
	e := errors.New("errortest")
	args1 := args{
		fs: []zapcore.Field{
			zap.Error(e),
			zap.String("test", "string"),
		},
	}
	want1 := sentry.ExtractStacktrace(e)
	args2 := args{
		fs: []zapcore.Field{
			zap.String("test", "string"),
			zap.Error(e),
		},
	}
	want2 := sentry.ExtractStacktrace(e)
	args3 := args{
		fs: []zapcore.Field{
			zap.String("test", "string"),
			zap.Error(e),
			zap.String("test", "string"),
		},
	}
	wantNot3 := sentry.ExtractStacktrace(e)
	args4 := args{
		fs: []zapcore.Field{
			zap.Error(fmt.Errorf("test")),
		},
	}
	wantNot4 := sentry.ExtractStacktrace(e)

	tests := []struct {
		name    string
		args    args
		want    *sentry.Stacktrace
		wantNot *sentry.Stacktrace
	}{
		{"fiest element is error", args1, want1, nil},
		{"last element is error", args2, want2, nil},
		{"mid element is error", args3, nil, wantNot3},
		{"error is not pkg/errors", args4, nil, wantNot4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTrace(tt.args.fs)
			if tt.want == nil {
				if reflect.DeepEqual(got, tt.wantNot) {
					t.Errorf("extractTrace() = %v, wantNot %v", got, tt.wantNot)
				}
			} else {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("extractTrace() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
