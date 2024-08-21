package types

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestResources_Add(t *testing.T) {
	tests := []struct {
		name string
		r1   Resources
		r2   Resources
		want Resources
	}{
		{
			name: "Add two resources",
			r1: Resources{
				CPU:  100,
				RAM:  512,
				Disk: 1024,
			},
			r2: Resources{
				CPU:  200,
				RAM:  1024,
				Disk: 2048,
			},
			want: Resources{
				CPU:  300,
				RAM:  1536,
				Disk: 3072,
			},
		},
		{
			name: "Add two resources with zero values",
			r1: Resources{
				CPU:  0,
				RAM:  0,
				Disk: 0,
			},
			r2: Resources{
				CPU:  0,
				RAM:  0,
				Disk: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r1.Add(tt.r2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResources_Subtract(t *testing.T) {
	tests := []struct {
		name    string
		r1      Resources
		r2      Resources
		want    Resources
		wantErr bool
		err     error
	}{
		{
			name: "Subtract two resources",
			r1: Resources{
				CPU:  200,
				RAM:  1024,
				Disk: 2048,
			},
			r2: Resources{
				CPU:  100,
				RAM:  512,
				Disk: 1024,
			},
			want: Resources{
				CPU:  100,
				RAM:  512,
				Disk: 1024,
			},
			wantErr: false,
		},
		{
			name: "Subtract two resources with zero values",
			r1: Resources{
				CPU:  0,
				RAM:  0,
				Disk: 0,
			},
			r2: Resources{
				CPU:  0,
				RAM:  0,
				Disk: 0,
			},
			want: Resources{
				CPU:  0,
				RAM:  0,
				Disk: 0,
			},
			wantErr: false,
		},
		{
			name: "Negative cpu value error",
			r1: Resources{
				CPU:  100,
				RAM:  512,
				Disk: 1024,
			},
			r2: Resources{
				CPU:  200,
				RAM:  512,
				Disk: 1024,
			},
			want:    Resources{},
			wantErr: true,
			err: &negativeValueError{
				resource: "CPU",
				r1:       float64(100),
				r2:       float64(200),
			},
		},
		{
			name: "Negative ram value error",
			r1: Resources{
				CPU:  200,
				RAM:  512,
				Disk: 1024,
			},
			r2: Resources{
				CPU:  200,
				RAM:  1024,
				Disk: 1024,
			},
			want:    Resources{},
			wantErr: true,
			err: &negativeValueError{
				resource: "RAM",
				r1:       uint64(512),
				r2:       uint64(1024),
			},
		},
		{
			name: "Negative disk value error",
			r1: Resources{
				CPU:  200,
				RAM:  1024,
				Disk: 1024,
			},
			r2: Resources{
				CPU:  200,
				RAM:  1024,
				Disk: 2048,
			},
			want:    Resources{},
			wantErr: true,
			err: &negativeValueError{
				resource: "Disk",
				r1:       uint64(1024),
				r2:       uint64(2048),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r1.Subtract(tt.r2)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.err, err)

			if tt.wantErr {
				assert.NotEmpty(t, err.Error())
			}
		})
	}
}
