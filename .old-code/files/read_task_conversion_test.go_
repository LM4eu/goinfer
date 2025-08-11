package files

import (
	"testing"

	"github.com/synw/goinfer/types"
)

func TestConvertTask_Float64ToIntConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		wantErr  bool
		errMsg   string
		expected *types.Task
	}{
		{
			name: "float64 ctx conversion",
			input: map[string]any{
				"name":     "test_task",
				"template": "test_template",
				"modelConf": []any{
					map[string]any{
						"name": "test_model",
						"ctx":  5.0, // float64 that should convert to int
					},
				},
			},
			wantErr: false,
			expected: &types.Task{
				Name:      "test_task",
				Template:  "test_template",
				ModelConf: types.ModelConf{Name: "test_model", Ctx: 5, GPULayers: 0},
				InferParams: types.InferenceParams{
					Threads:  4,   // default value
					NPredict: 512, // default value
					TopK:     40,  // default value
				},
			},
		},
		{
			name: "float64 gpu_layers conversion",
			input: map[string]any{
				"name":     "test_task",
				"template": "test_template",
				"modelConf": []any{
					map[string]any{
						"name":       "test_model",
						"gpu_layers": 3.0, // float64 that should convert to int
					},
				},
			},
			wantErr: false,
			expected: &types.Task{
				Name:      "test_task",
				Template:  "test_template",
				ModelConf: types.ModelConf{Name: "test_model", Ctx: 2048, GPULayers: 3}, // Ctx uses default value
				InferParams: types.InferenceParams{
					Threads:  4,   // default value
					NPredict: 512, // default value
					TopK:     40,  // default value
				},
			},
		},
		{
			name: "float64 threads conversion",
			input: map[string]any{
				"name":     "test_task",
				"template": "test_template",
				"inferParams": []any{
					map[string]any{
						"threads": 4.0, // float64 that should convert to int
					},
				},
			},
			wantErr: false,
			expected: &types.Task{
				Name:        "test_task",
				Template:    "test_template",
				ModelConf:   types.ModelConf{Ctx: 2048, GPULayers: 0},                   // default values
				InferParams: types.InferenceParams{Threads: 4, NPredict: 512, TopK: 40}, // other fields use default values
			},
		},
		{
			name: "float64 n_predict conversion",
			input: map[string]any{
				"name":     "test_task",
				"template": "test_template",
				"inferParams": []any{
					map[string]any{
						"n_predict": 100.0, // float64 that should convert to int
					},
				},
			},
			wantErr: false,
			expected: &types.Task{
				Name:        "test_task",
				Template:    "test_template",
				ModelConf:   types.ModelConf{Ctx: 2048, GPULayers: 0},                   // default values
				InferParams: types.InferenceParams{Threads: 4, NPredict: 100, TopK: 40}, // other fields use default values
			},
		},
		{
			name: "float64 top_k conversion",
			input: map[string]any{
				"name":     "test_task",
				"template": "test_template",
				"inferParams": []any{
					map[string]any{
						"top_k": 40.0, // float64 that should convert to int
					},
				},
			},
			wantErr: false,
			expected: &types.Task{
				Name:        "test_task",
				Template:    "test_template",
				ModelConf:   types.ModelConf{Ctx: 2048, GPULayers: 0},                   // default values
				InferParams: types.InferenceParams{Threads: 4, NPredict: 512, TopK: 40}, // other fields use default values
			},
		},
		{
			name: "float64 with fractional part should error",
			input: map[string]any{
				"name":     "test_task",
				"template": "test_template",
				"modelConf": []any{
					map[string]any{
						"name": "test_model",
						"ctx":  5.5, // float64 with fractional part
					},
				},
			},
			wantErr: true,
			errMsg:  "modelConf ctx must be an integer or a float64 without fractional part, got 5.500000",
		},
		{
			name: "invalid type for ctx should error",
			input: map[string]any{
				"name":     "test_task",
				"template": "test_template",
				"modelConf": []any{
					map[string]any{
						"name": "test_model",
						"ctx":  "invalid", // string instead of number
					},
				},
			},
			wantErr: true,
			errMsg:  "modelConf ctx must be an integer or float64, got string",
		},
		{
			name: "invalid type for threads should error",
			input: map[string]any{
				"name":     "test_task",
				"template": "test_template",
				"inferParams": []any{
					map[string]any{
						"threads": "invalid", // string instead of number
					},
				},
			},
			wantErr: true,
			errMsg:  "inferParams threads must be an integer or float64, got string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := convertTask(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("convertTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("convertTask() error message = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("convertTask() unexpected error = %v", err)
				return
			}

			// Compare the task fields
			if task.Name != tt.expected.Name {
				t.Errorf("convertTask() Name = %v, want %v", task.Name, tt.expected.Name)
			}
			if task.Template != tt.expected.Template {
				t.Errorf("convertTask() Template = %v, want %v", task.Template, tt.expected.Template)
			}
			if task.ModelConf.Name != tt.expected.ModelConf.Name {
				t.Errorf("convertTask() ModelConf.Name = %v, want %v", task.ModelConf.Name, tt.expected.ModelConf.Name)
			}
			if task.ModelConf.Ctx != tt.expected.ModelConf.Ctx {
				t.Errorf("convertTask() ModelConf.Ctx = %v, want %v", task.ModelConf.Ctx, tt.expected.ModelConf.Ctx)
			}
			if task.ModelConf.GPULayers != tt.expected.ModelConf.GPULayers {
				t.Errorf("convertTask() ModelConf.GPULayers = %v, want %v", task.ModelConf.GPULayers, tt.expected.ModelConf.GPULayers)
			}
			if task.InferParams.Threads != tt.expected.InferParams.Threads {
				t.Errorf("convertTask() InferParams.Threads = %v, want %v", task.InferParams.Threads, tt.expected.InferParams.Threads)
			}
			if task.InferParams.NPredict != tt.expected.InferParams.NPredict {
				t.Errorf("convertTask() InferParams.NPredict = %v, want %v", task.InferParams.NPredict, tt.expected.InferParams.NPredict)
			}
			if task.InferParams.TopK != tt.expected.InferParams.TopK {
				t.Errorf("convertTask() InferParams.TopK = %v, want %v", task.InferParams.TopK, tt.expected.InferParams.TopK)
			}
		})
	}
}
