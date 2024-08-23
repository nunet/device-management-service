package nunet

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/transform"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

type TransformTestSuite struct {
	suite.Suite
}

func TestTransformTestSuite(t *testing.T) {
	suite.Run(t, new(TransformTestSuite))
}

func (s *TransformTestSuite) TestTransformJobs() {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name: "Valid jobs map",
			input: map[string]any{
				"job1": map[string]any{"key": "value1"},
				"job2": map[string]any{"key": "value2"},
			},
			expected: []any{
				map[string]any{"name": "job1", "key": "value1"},
				map[string]any{"name": "job2", "key": "value2"},
			},
			wantErr: false,
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "Invalid input type",
			input:    "not a map",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := TransformJobs(nil, tt.input, tree.NewPath())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(transform.Normalize(tt.expected), transform.Normalize(result))
		})
	}
}

func (s *TransformTestSuite) TestTransformVolumes() {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name: "Valid volumes map",
			input: map[string]any{
				"vol1": map[string]any{"size": "10G"},
				"vol2": map[string]any{"size": "20G"},
			},
			expected: []any{
				map[string]any{"name": "vol1", "size": "10G"},
				map[string]any{"name": "vol2", "size": "20G"},
			},
			wantErr: false,
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "Invalid input type",
			input:    "not a map",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := TransformVolumes(nil, tt.input, tree.NewPath())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(transform.Normalize(tt.expected), transform.Normalize(result))
		})
	}
}

func (s *TransformTestSuite) TestTransformNetworks() {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name: "Valid networks map",
			input: map[string]any{
				"net1": map[string]any{"type": "subnet"},
				"net2": map[string]any{"type": "subnet"},
			},
			expected: []any{
				map[string]any{"name": "net1", "type": "subnet"},
				map[string]any{"name": "net2", "type": "subnet"},
			},
			wantErr: false,
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "Invalid input type",
			input:    "not a map",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := TransformNetworks(nil, tt.input, tree.NewPath())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(transform.Normalize(tt.expected), transform.Normalize(result))
		})
	}
}

func (s *TransformTestSuite) TestTransformExecution() {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name: "Valid execution map",
			input: map[string]any{
				"type":  "docker",
				"image": "ubuntu:latest",
				"cmd":   "echo hello",
			},
			expected: map[string]any{
				"type": "docker",
				"params": map[string]any{
					"image": "ubuntu:latest",
					"cmd":   "echo hello",
				},
			},
			wantErr: false,
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "Invalid input type",
			input:    "not a map",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := TransformExecution(nil, tt.input, tree.NewPath())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(transform.Normalize(tt.expected), transform.Normalize(result))
		})
	}
}

func (s *TransformTestSuite) TestTransformVolume() {
	tests := []struct {
		name     string
		input    any
		expected any
		root     map[string]any
		path     tree.Path
		wantErr  bool
	}{
		{
			name:  "Valid string input",
			input: "data:/mnt/data",
			expected: map[string]any{
				"name":       "data",
				"mountpoint": "/mnt/data",
			},
			wantErr: false,
		},
		{
			name: "Valid map input",
			input: map[string]any{
				"name":       "data",
				"mountpoint": "/mnt/data",
				"size":       "10G",
			},
			expected: map[string]any{
				"name":       "data",
				"mountpoint": "/mnt/data",
				"size":       "10G",
			},
			wantErr: false,
		},
		{
			name: "Inherit parent definition",
			root: map[string]any{
				"jobs": []map[string]any{
					{
						"name": "job1",
						"volumes": []map[string]any{
							{
								"name": "data",
								"type": "output",
							},
						},
					},
				},
			},
			input: "data:/mnt/data",
			path:  tree.NewPath("jobs", "[0]", "children", "[0]", "volumes", "[0]"),
			expected: map[string]any{
				"name":       "data",
				"type":       "output",
				"mountpoint": "/mnt/data",
			},
			wantErr: false,
		},
		{
			name:     "Invalid string input",
			input:    "invalid:format:volume",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Invalid input type",
			input:    123,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			r := tt.root
			result, err := TransformVolume(&r, tt.input, tt.path)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(transform.Normalize(tt.expected), transform.Normalize(result))
		})
	}
}

func (s *TransformTestSuite) TestTransformLibrary() {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name:  "Valid string input with version",
			input: "mylib:1.0.0",
			expected: map[string]any{
				"name":    "mylib",
				"version": "1.0.0",
			},
			wantErr: false,
		},
		{
			name:  "Valid string input without version",
			input: "mylib",
			expected: map[string]any{
				"name":    "mylib",
				"version": "",
			},
			wantErr: false,
		},
		{
			name: "Valid map input",
			input: map[string]any{
				"name":    "mylib",
				"version": "1.0.0",
			},
			expected: map[string]any{
				"name":    "mylib",
				"version": "1.0.0",
			},
			wantErr: false,
		},
		{
			name:     "Invalid string input",
			input:    "invalid:format:library",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Invalid input type",
			input:    123,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := TransformLibrary(nil, tt.input, tree.NewPath())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(transform.Normalize(tt.expected), transform.Normalize(result))
		})
	}
}

func (s *TransformTestSuite) TestTransformVolumeRemote() {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name: "Valid remote map",
			input: map[string]any{
				"type":     "ipfs",
				"endpoint": "localhost",
			},
			expected: map[string]any{
				"type": "ipfs",
				"params": map[string]any{
					"endpoint": "localhost",
				},
			},
			wantErr: false,
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "Invalid input type",
			input:    "not a map",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := TransformVolumeRemote(nil, tt.input, tree.NewPath())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(transform.Normalize(tt.expected), transform.Normalize(result))
		})
	}
}

func (s *TransformTestSuite) TestTransformNetwork() {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name: "Valid network map",
			input: map[string]any{
				"type": "subnet",
				"ports": []map[string]any{
					{"protocol": "tcp", "container_port": 8080, "host_port": 80},
					{"protocol": "udp", "container_port": 53, "host_port": 53},
				},
			},
			expected: map[string]any{
				"type": "subnet",
				"port_map": []map[string]any{
					{"protocol": "tcp", "container_port": 8080, "host_port": 80},
					{"protocol": "udp", "container_port": 53, "host_port": 53},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid network map with string ports",
			input: map[string]any{
				"type":  "subnet",
				"ports": []string{"8080", "80:8080", "udp:53:53"},
			},
			expected: map[string]any{
				"type": "subnet",
				"port_map": []map[string]any{
					{"protocol": "tcp", "container_port": 8080, "host_port": 8080},
					{"protocol": "tcp", "container_port": 8080, "host_port": 80},
					{"protocol": "udp", "container_port": 53, "host_port": 53},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid network map with integer ports",
			input: map[string]any{
				"type":  "subnet",
				"ports": []int{8080, 53},
			},
			expected: map[string]any{
				"type": "subnet",
				"port_map": []map[string]any{
					{"protocol": "tcp", "container_port": 8080, "host_port": 8080},
					{"protocol": "tcp", "container_port": 53, "host_port": 53},
				},
			},
			wantErr: false,
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "Invalid input type",
			input:    "not a map",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := TransformNetwork(nil, tt.input, tree.NewPath())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(transform.Normalize(tt.expected), transform.Normalize(result))
		})
	}
}

func (s *TransformTestSuite) TestNunetTransformer() {
	transformer := NewNuNetTransformer()

	tests := []struct {
		name     string
		input    map[string]any
		expected any
		wantErr  bool
	}{
		{
			name: "Valid input",
			input: map[string]any{
				"jobs": map[string]any{
					"job1": map[string]any{
						"volumes": map[string]any{
							"data": map[string]any{
								"type": "output",
								"remote": map[string]any{
									"type":     "ipfs",
									"endpoint": "localhost",
								},
							},
						},
						"children": map[string]any{
							"job2": map[string]any{
								"execution": map[string]any{
									"type":  "docker",
									"image": "ubuntu:latest",
									"cmd":   "echo hello",
								},
								"libraries": []string{
									"mylib:1.0.0",
								},
								"volumes": map[string]any{
									"vol1": "data:/mnt/data",
								},
								"networks": map[string]any{
									"net1": map[string]any{
										"type":  "subnet",
										"ports": []string{"80:8080", "udp:53:53"},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]any{
				"jobs": []map[string]any{
					{
						"name": "job1",
						"volumes": []map[string]any{
							{
								"name": "data",
								"type": "output",
								"remote": map[string]any{
									"type": "ipfs",
									"params": map[string]any{
										"endpoint": "localhost",
									},
								},
							},
						},
						"children": []map[string]any{
							{
								"name": "job2",
								"execution": map[string]any{
									"type": "docker",
									"params": map[string]any{
										"image": "ubuntu:latest",
										"cmd":   "echo hello",
									},
								},
								"libraries": []map[string]any{
									{
										"name":    "mylib",
										"version": "1.0.0",
									},
								},
								"volumes": []map[string]any{
									{
										"name":       "data",
										"type":       "output",
										"mountpoint": "/mnt/data",
										"remote": map[string]any{
											"type": "ipfs",
											"params": map[string]any{
												"endpoint": "localhost",
											},
										},
									},
								},
								"networks": []map[string]any{
									{
										"name": "net1",
										"type": "subnet",
										"port_map": []map[string]any{
											{"protocol": "tcp", "container_port": 8080, "host_port": 80},
											{"protocol": "udp", "container_port": 53, "host_port": 53},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			v := tt.input
			result, err := transformer.Transform(&v)
			if tt.wantErr {
				s.Error(err)
				return
			}

			s.NoError(err)
			s.Equal(transform.Normalize(tt.expected), transform.Normalize(result))
		})
	}
}
