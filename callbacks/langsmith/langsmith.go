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
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"log"
	"runtime/debug"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// Config 用于配置 LangsmithHandler
type Config struct {
	APIKey string
	APIURL string
}

// CallbackHandler 实现了 eino 的 Handler 接口
type CallbackHandler struct {
	cli Langsmith
}

// NewLangsmithHandler 创建一个新的 CallbackHandler
func NewLangsmithHandler(flowTrace *FlowTrace) (*CallbackHandler, error) {
	return &CallbackHandler{cli: flowTrace.cli}, nil
}

type LangsmithState struct {
	TraceID           string `json:"trace_id"`
	ParentRunID       string `json:"parent_run_id"`
	ParentDottedOrder string `json:"parent_dotted_order"`
}

type langsmithStateKey struct{}

func (c *CallbackHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info == nil {
		return ctx
	}

	ctx, state := GetOrInitState(ctx)
	runID := uuid.New().String()

	opts, _ := ctx.Value(langsmithTraceOptionKey{}).(*traceOptions)
	if opts == nil {
		opts = &traceOptions{}
	}
	in, err := sonic.MarshalString(input)
	if err != nil {
		log.Printf("marshal input error: %v, runinfo: %+v", err, info)
		return ctx
	}
	if state.TraceID == "" {
		state.TraceID = runID
	}
	run := &Run{
		ID:          runID,
		TraceID:     state.TraceID,
		Name:        runInfoToName(info),
		RunType:     runInfoToRunType(info),
		StartTime:   time.Now().UTC(),
		Inputs:      map[string]interface{}{"input": in},
		SessionName: opts.SessionName,
		Extra:       opts.Metadata,
	}
	if opts.ReferenceExampleID != "" {
		run.ReferenceExampleID = &opts.ReferenceExampleID
	}
	if state.ParentRunID != "" {
		run.ParentRunID = &state.ParentRunID
	}
	nowTime := run.StartTime.Format("20060102T150405000000")
	if state.ParentDottedOrder != "" {
		run.DottedOrder = fmt.Sprintf("%s.%sZ%s", state.ParentDottedOrder, nowTime, runID)
	} else {
		run.DottedOrder = fmt.Sprintf("%sZ%s", nowTime, runID)
	}

	err = c.cli.CreateRun(ctx, run)
	if err != nil {
		log.Printf("[langsmith] failed to create run: %v", err)
	}

	newState := &LangsmithState{
		TraceID:           state.TraceID,
		ParentRunID:       runID,
		ParentDottedOrder: run.DottedOrder,
	}
	return context.WithValue(ctx, langsmithStateKey{}, newState)
}

func (c *CallbackHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if info == nil {
		return ctx
	}
	state, ok := ctx.Value(langsmithStateKey{}).(*LangsmithState)
	if !ok || state == nil {
		log.Printf("[langsmith] no state in context on OnEnd, runinfo: %+v", info)
		return ctx
	}
	out, err := sonic.MarshalString(output)
	if err != nil {
		log.Printf("marshal output error: %v, runinfo: %+v", err, info)
		return ctx
	}

	endTime := time.Now().UTC()
	patch := &RunPatch{
		EndTime: &endTime,
		Outputs: map[string]interface{}{"output": out},
	}

	err = c.cli.UpdateRun(ctx, state.ParentRunID, patch)
	if err != nil {
		log.Printf("[langsmith] failed to update run: %v", err)
	}
	return ctx
}

func (c *CallbackHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info == nil {
		return ctx
	}
	state, ok := ctx.Value(langsmithStateKey{}).(*LangsmithState)
	if !ok || state == nil {
		log.Printf("[langsmith] no state in context on OnError, runinfo: %+v", info)
		return ctx
	}

	endTime := time.Now().UTC()
	errStr := err.Error()
	patch := &RunPatch{
		EndTime: &endTime,
		Error:   &errStr,
	}

	updateErr := c.cli.UpdateRun(ctx, state.ParentRunID, patch)
	if updateErr != nil {
		log.Printf("[langsmith] failed to update run with error: %v", updateErr)
	}
	return ctx
}

// OnStartWithStreamInput 处理流式输入
func (c *CallbackHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	if info == nil {
		input.Close()
		return ctx
	}

	// 首先创建 run，然后用流式输入更新它
	ctx, state := GetOrInitState(ctx)
	runID := uuid.New().String()

	opts, _ := ctx.Value(langsmithTraceOptionKey{}).(*traceOptions)
	if opts == nil {
		opts = &traceOptions{}
	}
	if state.TraceID == "" {
		state.TraceID = runID
	}

	run := &Run{
		ID:        runID,
		TraceID:   state.TraceID,
		Name:      runInfoToName(info),
		RunType:   runInfoToRunType(info),
		StartTime: time.Now().UTC(),
		//Inputs:      map[string]interface{}{"stream_inputs": inMessage}, // 初始为空
		SessionName: opts.SessionName,
		//Extra:       extra,
	}
	nowTime := run.StartTime.Format("20060102T150405000000")
	if state.ParentDottedOrder != "" {
		run.DottedOrder = fmt.Sprintf("%s.%sZ%s", state.ParentDottedOrder, nowTime, runID)
	} else {
		run.DottedOrder = fmt.Sprintf("%sZ%s", nowTime, runID)
	}

	// 启动一个 goroutine 来处理输入流
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[langsmith] recovered in OnStartWithStreamInput: %v\n%s", r, debug.Stack())
			}
			input.Close()
		}()

		var inputs []callbacks.CallbackInput
		for {
			chunk, err := input.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("[langsmith] error receiving stream input: %v", err)
				break
			}
			inputs = append(inputs, chunk)
		}
		modelConf, inMessage, extra, err_ := extractModelInput(convModelCallbackInput(inputs))
		if err_ != nil {
			log.Printf("extract stream model input error: %v, runinfo: %+v", err_, info)
			return
		}
		if extra == nil {
			extra = map[string]interface{}{}
		}
		if modelConf != nil {
			extra["model_conf"] = modelConf
		}

		if opts.ReferenceExampleID != "" {
			run.ReferenceExampleID = &opts.ReferenceExampleID
		}
		if state.ParentRunID != "" {
			run.ParentRunID = &state.ParentRunID
		}

		run.Inputs = map[string]interface{}{"stream_inputs": inMessage}
		run.Extra = extra
		err := c.cli.CreateRun(ctx, run)
		if err != nil {
			log.Printf("[langsmith] failed to create run for stream: %v", err)
		}
	}()

	newState := &LangsmithState{
		TraceID:           state.TraceID,
		ParentRunID:       runID,
		ParentDottedOrder: run.DottedOrder,
	}
	return context.WithValue(ctx, langsmithStateKey{}, newState)
}

// OnEndWithStreamOutput 处理流式输出
func (c *CallbackHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	if info == nil {
		output.Close()
		return ctx
	}
	state, ok := ctx.Value(langsmithStateKey{}).(*LangsmithState)
	if !ok || state == nil {
		log.Printf("[langsmith] no state in context on OnEndWithStreamOutput, runinfo: %+v", info)
		output.Close()
		return ctx
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[langsmith] recovered in OnEndWithStreamOutput: %v\n%s", r, debug.Stack())
			}
			output.Close()
		}()

		var outputs []callbacks.CallbackOutput
		for {
			chunk, err := output.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("[langsmith] error receiving stream output: %v", err)
				break
			}
			outputs = append(outputs, chunk)
		}
		usage, outMessage, extra, err_ := extractModelOutput(convModelCallbackOutput(outputs))
		if err_ != nil {
			log.Printf("extract stream model output error: %v, runinfo: %+v", err_, info)
			return
		}
		if extra == nil {
			extra = map[string]interface{}{}
		}
		if usage != nil {
			extra["model_usage"] = usage
		}
		endTime := time.Now().UTC()
		patch := &RunPatch{
			EndTime: &endTime,
			Outputs: map[string]interface{}{"stream_outputs": outMessage},
			Extra:   extra,
		}

		// 使用后台 context
		err := c.cli.UpdateRun(context.Background(), state.ParentRunID, patch)
		if err != nil {
			log.Printf("[langsmith] failed to update run with stream output: %v", err)
		}
	}()

	return ctx
}
