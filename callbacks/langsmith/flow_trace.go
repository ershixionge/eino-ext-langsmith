package langsmith

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"log"
	"time"
)

type FlowTrace struct {
	cli Langsmith
}

func NewFlowTrace(cfg *Config) *FlowTrace {
	cli := NewLangsmith(cfg.APIKey, cfg.APIURL)
	return &FlowTrace{cli: cli}
}

func (ft *FlowTrace) StartSpan(ctx context.Context, name, spanType string, state *LangsmithState) (context.Context, string, error) {
	opts, _ := ctx.Value(langsmithTraceOptionKey{}).(*traceOptions)
	if opts == nil {
		opts = &traceOptions{}
	}
	if state == nil {
		state = &LangsmithState{}
	}
	runID := uuid.New().String()
	if state.traceID == "" {
		state.traceID = runID
	}
	run := &Run{
		ID:          runID,
		TraceID:     state.traceID,
		Name:        name,
		RunType:     RunType(spanType),
		StartTime:   time.Now().UTC(),
		SessionName: opts.SessionName,
		Extra:       opts.Metadata,
	}
	if opts.ReferenceExampleID != "" {
		run.ReferenceExampleID = &opts.ReferenceExampleID
	}
	if state.parentRunID != "" {
		run.ParentRunID = &state.parentRunID
	}
	nowTime := run.StartTime.Format("20060102T150405000000")
	if state.parentDottedOrder != "" {
		run.DottedOrder = fmt.Sprintf("%s.%sZ%s", state.parentDottedOrder, nowTime, runID)
	} else {
		run.DottedOrder = fmt.Sprintf("%sZ%s", nowTime, runID)
	}
	err := ft.cli.CreateRun(ctx, run)
	if err != nil {
		return nil, "", err
	}
	newState := &LangsmithState{
		traceID:           state.traceID,
		parentRunID:       runID,
		parentDottedOrder: run.DottedOrder,
	}

	return context.WithValue(ctx, langsmithStateKey{}, newState), runID, nil
}

func (ft *FlowTrace) FinishSpan(ctx context.Context, runID string) {
	endTime := time.Now().UTC()
	patch := &RunPatch{
		EndTime: &endTime,
	}

	err := ft.cli.UpdateRun(ctx, runID, patch)
	if err != nil {
		log.Printf("[langsmith] failed to FinishSpan: %v", err)
	}
}

func (ft *FlowTrace) SpanToString(ctx context.Context) (string, error) {
	ctx, state := GetState(ctx)
	if state == nil {
		return "", nil
	}
	val, err := sonic.Marshal(state)
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func (ft *FlowTrace) StringToSpan(val string) (*LangsmithState, error) {
	if val == "" {
		return nil, nil
	}
	state := &LangsmithState{}
	err := sonic.Unmarshal([]byte(val), state)
	return state, err
}
