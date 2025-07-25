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
	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/libs/acl/langsmith"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
)

func runInfoToName(info *callbacks.RunInfo) string {
	if len(info.Name) != 0 {
		return info.Name
	}
	return info.Type + string(info.Component)
}

func runInfoToRunType(info *callbacks.RunInfo) langsmith.RunType {
	switch info.Component {
	case components.ComponentOfChatModel:
		return langsmith.RunTypeLLM
	case components.ComponentOfTool:
		return langsmith.RunTypeTool
	default:
		return langsmith.RunTypeChain
	}
}

func streamInputsToMap(inputs []callbacks.CallbackInput) map[string]interface{} {
	if len(inputs) == 0 {
		return nil
	}
	// 为简化起见，我们序列化整个切片。
	// 更复杂的实现可以合并配置和消息。
	b, err := sonic.Marshal(inputs)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	var data []map[string]interface{}
	if sonic.Unmarshal(b, &data) != nil {
		return map[string]interface{}{"raw": string(b)}
	}
	return map[string]interface{}{"stream_inputs": data}
}

func streamOutputsToMap(outputs []callbacks.CallbackOutput) map[string]interface{} {
	if len(outputs) == 0 {
		return nil
	}
	// 为简化起见，我们序列化整个切片。
	// 更复杂的实现可以聚合消息和 token 使用情况。
	b, err := sonic.Marshal(outputs)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	var data []map[string]interface{}
	if sonic.Unmarshal(b, &data) != nil {
		return map[string]interface{}{"raw": string(b)}
	}
	return map[string]interface{}{"stream_outputs": data}
}

func inputToMap(input callbacks.CallbackInput) map[string]interface{} {
	if input == nil {
		return nil
	}
	// 尝试将输入转换为模型输入
	if mcbi := model.ConvCallbackInput(input); mcbi != nil {
		return map[string]interface{}{
			"messages": mcbi.Messages,
			"config":   mcbi.Config,
			"extra":    mcbi.Extra,
		}
	}
	// 否则，进行通用序列化
	b, err := sonic.Marshal(input)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"raw": string(b)}
}

func outputToMap(output callbacks.CallbackOutput) map[string]interface{} {
	if output == nil {
		return nil
	}
	// 尝试将输出转换为模型输出
	if mcbo := model.ConvCallbackOutput(output); mcbo != nil {
		return map[string]interface{}{
			"message":    mcbo.Message,
			"tokenUsage": mcbo.TokenUsage,
			"extra":      mcbo.Extra,
		}
	}
	// 否则，进行通用序列化
	b, err := sonic.Marshal(output)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"raw": string(b)}
}
