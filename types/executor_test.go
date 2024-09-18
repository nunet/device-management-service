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

func TestExecutor_Comparable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		e    *Executor
		r    Executor
		want Comparison
	}{
		{
			name: "Equal",
			e: &Executor{
				ExecutorType: ExecutorTypeDocker,
			},
			r: Executor{
				ExecutorType: ExecutorTypeDocker,
			},
			want: Equal,
		},
		{
			name: "None",
			e: &Executor{
				ExecutorType: ExecutorTypeDocker,
			},
			r: Executor{
				ExecutorType: ExecutorTypeFirecracker,
			},
			want: None,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.e.Compare(tt.r)
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
			l: Executors{
				Executor{
					ExecutorType: ExecutorTypeDocker,
				},
			},
			r: Executors{
				Executor{
					ExecutorType: ExecutorTypeDocker,
				},
			},
			want: Equal,
		},
		{
			name: "Worse",
			l: Executors{
				Executor{
					ExecutorType: ExecutorTypeDocker,
				},
			},
			r: Executors{
				Executor{
					ExecutorType: ExecutorTypeDocker,
				},
				Executor{
					ExecutorType: ExecutorTypeFirecracker,
				},
			},
			want: Worse,
		},
		{
			name: "Better",
			l: Executors{
				Executor{
					ExecutorType: ExecutorTypeDocker,
				},
				Executor{
					ExecutorType: ExecutorTypeFirecracker,
				},
			},
			r: Executors{
				Executor{
					ExecutorType: ExecutorTypeDocker,
				},
			},
			want: Better,
		},
		{
			name: "None",
			l: Executors{
				Executor{
					ExecutorType: ExecutorTypeDocker,
				},
			},
			r: Executors{
				Executor{
					ExecutorType: ExecutorTypeFirecracker,
				},
			},
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
				l: Executors{
					Executor{
						ExecutorType: ExecutorTypeDocker,
					},
				},
				r: Executors{
					Executor{
						ExecutorType: ExecutorTypeFirecracker,
					},
				},
				want: Executors{
					Executor{
						ExecutorType: ExecutorTypeDocker,
					},
					Executor{
						ExecutorType: ExecutorTypeFirecracker,
					},
				},
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
				l: Executors{
					Executor{
						ExecutorType: ExecutorTypeDocker,
					},
					Executor{
						ExecutorType: ExecutorTypeFirecracker,
					},
				},
				r: Executors{
					Executor{
						ExecutorType: ExecutorTypeFirecracker,
					},
				},
				want: Executors{
					Executor{
						ExecutorType: ExecutorTypeDocker,
					},
				},
			},
			{
				name: "Subtract all",
				l: Executors{
					Executor{
						ExecutorType: ExecutorTypeDocker,
					},
				},
				r: Executors{
					Executor{
						ExecutorType: ExecutorTypeDocker,
					},
				},
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
		executor Executor
		list     Executors
		want     bool
	}{
		{
			name: "Contains",
			executor: Executor{
				ExecutorType: ExecutorTypeDocker,
			},
			list: Executors{
				Executor{
					ExecutorType: ExecutorTypeDocker,
				},
			},
			want: true,
		},
		{
			name: "!Contains",
			executor: Executor{
				ExecutorType: ExecutorTypeFirecracker,
			},
			list: Executors{
				Executor{
					ExecutorType: ExecutorTypeDocker,
				},
			},
			want: false,
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

func TestExecutor_Equal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		executor1 Executor
		executor2 Executor
		want      bool
	}{
		{
			name: "Equal",
			executor1: Executor{
				ExecutorType: ExecutorTypeDocker,
			},
			executor2: Executor{
				ExecutorType: ExecutorTypeDocker,
			},
			want: true,
		},
		{
			name: "!Equal",
			executor1: Executor{
				ExecutorType: ExecutorTypeFirecracker,
			},
			executor2: Executor{
				ExecutorType: ExecutorTypeDocker,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.executor1.Equal(tt.executor2); got != tt.want {
				t.Errorf("Executor.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
