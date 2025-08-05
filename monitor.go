package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/UTC-Six/pool/threading"
)

// ContextEnhancer 增强context的函数类型
type ContextEnhancer func(ctx context.Context) context.Context

// TrackerOption 追踪器选项
type TrackerOption func(*LatencyTracker)

// WithLogger 设置日志函数选项
func WithLogger(logger func(ctx context.Context, format string, args ...interface{})) TrackerOption {
	return func(lt *LatencyTracker) {
		lt.logger = logger
	}
}

// WithContextEnhancer 设置context增强函数选项
func WithContextEnhancer(enhancer ContextEnhancer) TrackerOption {
	return func(lt *LatencyTracker) {
		lt.contextEnhancer = enhancer
	}
}

// 默认追踪器实例
var defaultTracker *LatencyTracker

// LatencyTracker 延迟追踪器
type LatencyTracker struct {
	logger          func(ctx context.Context, format string, args ...interface{})
	contextEnhancer ContextEnhancer
}

// NewLatencyTracker 创建延迟追踪器
func NewLatencyTracker(opts ...TrackerOption) *LatencyTracker {
	lt := &LatencyTracker{
		logger:          defaultLogger,          // 使用默认logger
		contextEnhancer: defaultContextEnhancer, // 使用默认的context增强器
	}

	// 应用选项
	for _, opt := range opts {
		opt(lt)
	}

	return lt
}

// defaultLogger 默认日志函数
func defaultLogger(ctx context.Context, format string, args ...interface{}) {
	// 默认使用标准log，可以通过WithLogger覆盖
	fmt.Printf(format+"\n", args...)
}

// defaultContextEnhancer 默认的context增强器
func defaultContextEnhancer(ctx context.Context) context.Context {
	// 如果没有设置WithContextEnhancer，直接返回Background
	return context.Background()
}

// Track 追踪执行时间（最优实现）
func Track(ctx context.Context, startTime time.Time, name string, logger func(ctx context.Context, format string, args ...interface{})) {
	// 确保defaultTracker已初始化
	if defaultTracker == nil {
		defaultTracker = NewLatencyTracker()
	}

	// 如果没有提供logger，使用默认的
	if logger == nil {
		logger = defaultTracker.logger
	}

	// 计算耗时
	duration := time.Since(startTime)

	// 使用配置的context增强器创建新的context
	enhancedCtx := defaultTracker.contextEnhancer(ctx)

	// 使用 threading.GoSafe 异步记录完成日志（最优选择）
	threading.GoSafe(func() error {
		// 使用 logz 格式的日志，traceID 会自动从 enhancedCtx 中提取
		logger(enhancedCtx, "[Latency] Name=%s, Duration=%v, Status=completed", name, duration)
		return nil
	}, threading.WithTag(name),
		threading.WithLog(func(format string, args ...interface{}) {
			// 如果 threading 内部需要日志，也使用我们的 logger
			logger(enhancedCtx, format, args...)
		}),
		threading.WithRecovery(func(r interface{}) {
			logger(enhancedCtx, "[Latency] Name=%s, Duration=%v, Status=logError, Error=%v", name, duration, r)
		}))
}
