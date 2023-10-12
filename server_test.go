package orchestra

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestNewServerPlayer(t *testing.T) {
	type args struct {
		opts []ServerPlayerOption
	}
	tests := []struct {
		name string
		args args
		want *ServerPlayer
	}{
		{
			name: "default",
			args: args{},
			want: &ServerPlayer{Server: &http.Server{}, ShutdownTimeout: time.Second * 10},
		},
		{
			name: "set timeout to 5s",
			args: args{opts: []ServerPlayerOption{
				WithShutdownTimeout(time.Second * 5),
			}},
			want: &ServerPlayer{Server: &http.Server{}, ShutdownTimeout: time.Second * 5},
		},
		{
			name: "replace default http server",
			args: args{opts: []ServerPlayerOption{
				WithHTTPServer(&http.Server{Addr: ":4321"}),
			}},
			want: &ServerPlayer{Server: &http.Server{Addr: ":4321"}, ShutdownTimeout: time.Second * 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewServerPlayer(tt.args.opts...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewServerPlayer() = %v, want %v", got, tt.want)
			}
		})
	}
}
