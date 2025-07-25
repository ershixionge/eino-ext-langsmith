/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package langsmith

import (
	"context"
	"time"
)

// Langsmith 定义了与 Langsmith API 交互的客户端接口
type Langsmith interface {
	CreateRun(ctx context.Context, run *Run) error
	UpdateRun(ctx context.Context, runID string, patch *RunPatch) error
}

// RunType 定义了 run 的类型
type RunType string

const (
	RunTypeChain RunType = "chain"
	RunTypeLLM   RunType = "llm"
	RunTypeTool  RunType = "tool"
)

// Run 代表一个 Langsmith run
type Run struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	RunType            RunType                `json:"run_type"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            *time.Time             `json:"end_time,omitempty"`
	Inputs             map[string]interface{} `json:"inputs"`
	Outputs            map[string]interface{} `json:"outputs,omitempty"`
	Error              *string                `json:"error,omitempty"`
	ParentRunID        *string                `json:"parent_run_id,omitempty"`
	TraceID            string                 `json:"trace_id"`
	Extra              map[string]interface{} `json:"extra,omitempty"`
	ProjectName        string                 `json:"project_name,omitempty"`
	ReferenceExampleID *string                `json:"reference_example_id,omitempty"`
}

// RunPatch 用于更新一个 run
type RunPatch struct {
	EndTime *time.Time             `json:"end_time,omitempty"`
	Outputs map[string]interface{} `json:"outputs,omitempty"`
	Error   *string                `json:"error,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

// NewLangsmith 是一个占位符，实际实现中需要连接 Langsmith API
// 这里我们返回一个 nil 客户端，因为在测试中我们会 mock 它
func NewLangsmith(apiKey, apiUrl string) Langsmith {
	// 实际的实现会在这里初始化一个 HTTP 客户端来调用 Langsmith API
	// 例如:
	// return &langsmithClient{
	//     apiKey: apiKey,
	//     baseURL: apiUrl,
	//     httpClient: &http.Client{Timeout: 10 * time.Second},
	// }
	return nil
}
