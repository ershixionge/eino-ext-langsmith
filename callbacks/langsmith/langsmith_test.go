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
	"github.com/ershixionge/eino-ext-langsmith/callbacks/langsmith/mock"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/mockey"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// TestLangsmithCallbackDemo 演示了如何集成和使用 Langsmith 回调
func TestLangsmithCallbackDemo(t *testing.T) {
	// 1. 初始化 mock 控制器和 mock 客户端
	ctrl := gomock.NewController(t)
	mockClient := mock.NewMockLangsmith(ctrl)

	// 2. 使用 mockey 来 patch NewLangsmithHandler 的创建过程，使其返回我们的 mock 客户端
	// 这样我们就可以在不实际调用 Langsmith API 的情况下进行测试
	mockey.PatchConvey("Test Langsmith Callback Demo", t, func() {
		mockey.Mock(NewLangsmith).Return(mockClient).Build()

		// 3. 创建 Langsmith 回调处理器
		cbh, err := NewLangsmithHandler(&Config{
			APIKey: "test-key",
		})
		assert.NoError(t, err)

		// 4. 将处理器设置为 Eino 的全局回调
		callbacks.InitCallbackHandlers([]callbacks.Handler{cbh})

		// 5. (可选) 使用 SetTrace 在 context 中设置 Trace 级别的属性
		ctx := context.Background()
		ctx = SetTrace(ctx,
			WithMetadata(map[string]interface{}{"user": "test-user"}),
		)

		// 6. 构建一个 Eino Graph
		g := compose.NewGraph[string, string]()
		_ = g.AddLambdaNode("node1", compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
			return "output1", nil
		}), compose.WithNodeName("node1"))
		_ = g.AddLambdaNode("node2", compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
			return strings.Repeat(input, 2), nil
		}), compose.WithNodeName("node2"))
		_ = g.AddEdge(compose.START, "node1")
		_ = g.AddEdge("node1", "node2")
		_ = g.AddEdge("node2", compose.END)

		runner, err := g.Compile(ctx)
		assert.NoError(t, err)

		// 7. 设置 mock 期望
		// 我们期望有 3 个 run 被创建 (graph, node1, node2)
		// 并且它们都被正确地更新
		mockClient.EXPECT().CreateRun(gomock.Any(), gomock.Any()).Times(3)
		mockClient.EXPECT().UpdateRun(gomock.Any(), gomock.Any(), gomock.Any()).Times(3)

		// 8. 执行 Graph
		result, err := runner.Invoke(ctx, "input")
		assert.NoError(t, err)
		assert.Equal(t, "output1output1", result)
	})
}

func TestLangsmithCallback_Stream(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock.NewMockLangsmith(ctrl)

	mockey.PatchConvey("Test Langsmith Callback Stream", t, func() {
		mockey.Mock(NewLangsmith).Return(mockClient).Build()

		cbh, err := NewLangsmithHandler(&Config{APIKey: "test-key"})
		assert.NoError(t, err)

		ctx := context.Background()

		// Mock 期望
		// CreateRun 在流开始时被调用一次
		mockClient.EXPECT().CreateRun(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, run *Run) error {
			assert.Equal(t, "My-Streaming-Project", run.SessionName)
			return nil
		}).Times(1)
		// UpdateRun 被调用两次：一次用于输入，一次用于输出
		mockClient.EXPECT().UpdateRun(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

		// 设置 trace 信息
		ctx = SetTrace(ctx, WithSessionName("My-Streaming-Project"))

		// 准备流式输入
		insr, insw := schema.Pipe[callbacks.CallbackInput](1)
		go func() {
			defer insw.Close()
			_ = insw.Send(&model.CallbackInput{Messages: []*schema.Message{{Role: schema.User, Content: "Hello"}}}, nil)
		}()

		// 准备流式输出
		outsr, outsw := schema.Pipe[callbacks.CallbackOutput](1)
		go func() {
			defer outsw.Close()
			_ = outsw.Send(&model.CallbackOutput{Message: &schema.Message{Role: schema.Assistant, Content: "Hi there!"}}, nil)
		}()

		runInfo := &callbacks.RunInfo{Component: components.ComponentOfChatModel, Name: "StreamingChat"}

		// 执行
		ctx = cbh.OnStartWithStreamInput(ctx, runInfo, insr)
		cbh.OnEndWithStreamOutput(ctx, runInfo, outsr)

		// 等待 goroutine 完成。在实际测试中，可能需要同步机制。
		// 对于这个演示，短暂的睡眠就足够了。
		time.Sleep(100 * time.Millisecond)
	})
}
