package monitoring

import (
	"context"
	"time"
)

type HeartbeatFunc func() error
type MonitorConfig struct {
	InitialDelay int    `json:"initialDelay"`
	Interval     int    `json:"interval"`
	Id           string `json:"id"`
	DisplayName  string `json:"displayName"`
}

type MonitoringAgent struct {
	heartbeat HeartbeatFunc
	started   bool
	Healthy   bool
	MonitorConfig
}

func NewAgent(ctx context.Context, f HeartbeatFunc, config MonitorConfig) *MonitoringAgent {
	agent := &MonitoringAgent{
		MonitorConfig: config,
		heartbeat:     f,
	}
	agent.start(ctx)
	return agent
}

func (m *MonitoringAgent) start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	go func() {
		defer cancel()
		time.AfterFunc(time.Duration(m.InitialDelay), func() {
			ticker := time.NewTicker(time.Duration(m.Interval))
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					err := m.heartbeat()
					m.Healthy = err == nil
				case <-ctx.Done():
					m.started = false
					return
				}
			}
		})
	}()
	m.started = true
}

// func (m *MonitoringAgent)
