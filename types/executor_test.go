// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutorType_Comparable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		l    ExecutorType
		r    ExecutorType
		want Comparison
	}{
		{
			name: "Equal",
			l:    ExecutorTypeDocker,
			r:    ExecutorTypeDocker,
			want: Equal,
		},
		{
			name: "None",
			l:    ExecutorTypeDocker,
			r:    ExecutorTypeFirecracker,
			want: None,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.l.Compare(tt.r)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestExecutors_Comparable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		l    Executors
		r    Executors
		want Comparison
	}{
		{
			name: "Equal",
			l:    Executors{ExecutorTypeDocker},
			r:    Executors{ExecutorTypeDocker},
			want: Equal,
		},
		{
			name: "Worse",
			l:    Executors{ExecutorTypeDocker},
			r:    Executors{ExecutorTypeDocker, ExecutorTypeFirecracker},
			want: Worse,
		},
		{
			name: "Better",
			l:    Executors{ExecutorTypeDocker, ExecutorTypeFirecracker},
			r:    Executors{ExecutorTypeDocker},
			want: Better,
		},
		{
			name: "None",
			l:    Executors{ExecutorTypeDocker},
			r:    Executors{ExecutorTypeFirecracker},
			want: None,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.l.Compare(tt.r)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestExecutors_Calculable(t *testing.T) {
	t.Parallel()

	t.Run("Add checks", func(t *testing.T) {
		tests := []struct {
			name string
			l    Executors
			r    Executors
			want Executors
		}{
			{
				name: "Add",
				l:    Executors{ExecutorTypeDocker},
				r:    Executors{ExecutorTypeFirecracker},
				want: Executors{ExecutorTypeDocker, ExecutorTypeFirecracker},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Add(tt.r); err != nil {
					t.Errorf("Executors.Add() = %v", err)
				}
				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("Executors.Add() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})

	t.Run("Subtract checks", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			l    Executors
			r    Executors
			want Executors
		}{
			{
				name: "Subtract",
				l:    Executors{ExecutorTypeDocker, ExecutorTypeFirecracker},
				r: Executors{
					ExecutorTypeFirecracker,
				},
				want: Executors{ExecutorTypeDocker},
			},
			{
				name: "Subtract all",
				l:    Executors{ExecutorTypeDocker},
				r:    Executors{ExecutorTypeDocker},
				want: Executors{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.l.Subtract(tt.r); err != nil {
					t.Errorf("Executors.Subtract() = %v", err)
				}
				if !reflect.DeepEqual(tt.l, tt.want) {
					t.Errorf("Executors.Subtract() = %v, want %v", tt.l, tt.want)
				}
			})
		}
	})
}

func TestExecutors_Contains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		executor ExecutorType
		list     Executors
		want     bool
	}{
		{
			name:     "Contains",
			executor: ExecutorTypeDocker,
			list:     Executors{ExecutorTypeDocker},
			want:     true,
		},
		{
			name:     "!Contains",
			executor: ExecutorTypeFirecracker,
			list:     Executors{ExecutorTypeDocker},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.list.Contains(tt.executor); got != tt.want {
				t.Errorf("Executors.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
