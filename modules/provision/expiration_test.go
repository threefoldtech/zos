package provision

import (
	"testing"
	"time"
)

func TestIsExpired(t *testing.T) {
	type args struct {
		r *Reservation
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "expired",
			args: args{&Reservation{
				Created:  time.Now().Add(-time.Minute),
				Duration: time.Second,
			}},
			want: true,
		},
		{
			name: "not expired",
			args: args{&Reservation{
				Created:  time.Now(),
				Duration: time.Minute,
			}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExpired(tt.args.r); got != tt.want {
				t.Errorf("isExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
