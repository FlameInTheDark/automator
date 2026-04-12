package plugins

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/FlameInTheDark/emerald/pkg/pluginapi"
)

type TriggerEventHandler func(ctx context.Context, subscription pluginapi.TriggerSubscription, event *pluginapi.TriggerEvent) error

type TriggerRuntimeService struct {
	manager *Manager
	onEvent TriggerEventHandler

	mu            sync.RWMutex
	runtimes      map[string]*activeTriggerRuntime
	subscriptions map[string]pluginapi.TriggerSubscription
}

type activeTriggerRuntime struct {
	runtime   pluginapi.TriggerRuntime
	cancel    context.CancelFunc
	release   func()
	closeOnce sync.Once
}

func NewTriggerRuntimeService(manager *Manager, onEvent TriggerEventHandler) *TriggerRuntimeService {
	return &TriggerRuntimeService{
		manager:       manager,
		onEvent:       onEvent,
		runtimes:      make(map[string]*activeTriggerRuntime),
		subscriptions: make(map[string]pluginapi.TriggerSubscription),
	}
}

func (s *TriggerRuntimeService) Reload(ctx context.Context, subscriptions []pluginapi.TriggerSubscription) error {
	if s == nil || s.manager == nil {
		return nil
	}

	grouped := make(map[string][]pluginapi.TriggerSubscription)
	index := make(map[string]pluginapi.TriggerSubscription, len(subscriptions))

	for _, subscription := range subscriptions {
		kind, pluginID, nodeID, ok := ParseNodeType(subscription.NodeType)
		if !ok || kind != pluginapi.NodeKindTrigger {
			return errors.New("plugin trigger subscription uses an invalid node type")
		}

		subscription.SubscriptionID = strings.TrimSpace(subscription.SubscriptionID)
		subscription.NodeID = strings.TrimSpace(subscription.NodeID)
		subscription.NodeType = strings.TrimSpace(subscription.NodeType)
		subscription.NodeInstanceID = strings.TrimSpace(subscription.NodeInstanceID)
		subscription.PipelineID = strings.TrimSpace(subscription.PipelineID)

		if subscription.SubscriptionID == "" {
			return errors.New("plugin trigger subscription id is required")
		}
		if subscription.NodeID == "" {
			subscription.NodeID = nodeID
		}

		grouped[pluginID] = append(grouped[pluginID], subscription)
		index[subscription.SubscriptionID] = subscription
	}

	s.mu.Lock()
	s.subscriptions = index

	for pluginID, runtime := range s.runtimes {
		if _, ok := grouped[pluginID]; ok {
			continue
		}
		runtime.close()
		delete(s.runtimes, pluginID)
	}

	var errs []error
	for pluginID, pluginSubscriptions := range grouped {
		runtime, err := s.ensureRuntimeLocked(pluginID)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		snapshot := pluginapi.TriggerSubscriptionSnapshot{
			Subscriptions: append([]pluginapi.TriggerSubscription(nil), pluginSubscriptions...),
		}
		if err := runtime.runtime.SendSnapshot(ctx, snapshot); err != nil {
			runtime.close()
			delete(s.runtimes, pluginID)

			runtime, reopenErr := s.ensureRuntimeLocked(pluginID)
			if reopenErr != nil {
				errs = append(errs, reopenErr)
				continue
			}
			if err := runtime.runtime.SendSnapshot(ctx, snapshot); err != nil {
				errs = append(errs, err)
				runtime.close()
				delete(s.runtimes, pluginID)
			}
		}
	}
	s.mu.Unlock()

	return errors.Join(errs...)
}

func (s *TriggerRuntimeService) Stop() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for pluginID, runtime := range s.runtimes {
		runtime.close()
		delete(s.runtimes, pluginID)
	}
	s.subscriptions = make(map[string]pluginapi.TriggerSubscription)
}

func (s *TriggerRuntimeService) ensureRuntimeLocked(pluginID string) (*activeTriggerRuntime, error) {
	if runtime := s.runtimes[pluginID]; runtime != nil {
		return runtime, nil
	}

	runtime, release, err := s.manager.OpenTriggerRuntime(context.Background(), pluginID)
	if err != nil {
		return nil, err
	}

	runtimeCtx, cancel := context.WithCancel(context.Background())
	active := &activeTriggerRuntime{
		runtime: runtime,
		cancel:  cancel,
		release: release,
	}
	s.runtimes[pluginID] = active

	go s.listenRuntime(runtimeCtx, pluginID, active)

	return active, nil
}

func (s *TriggerRuntimeService) listenRuntime(ctx context.Context, pluginID string, active *activeTriggerRuntime) {
	defer func() {
		active.close()

		s.mu.Lock()
		if current := s.runtimes[pluginID]; current == active {
			delete(s.runtimes, pluginID)
		}
		s.mu.Unlock()
	}()

	for {
		event, err := active.runtime.Recv(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				return
			}
			return
		}
		if event == nil || strings.TrimSpace(event.SubscriptionID) == "" {
			continue
		}

		s.mu.RLock()
		subscription, ok := s.subscriptions[strings.TrimSpace(event.SubscriptionID)]
		handler := s.onEvent
		s.mu.RUnlock()
		if !ok || handler == nil {
			continue
		}

		_ = handler(context.Background(), subscription, event)
	}
}

func (r *activeTriggerRuntime) close() {
	if r == nil {
		return
	}

	r.closeOnce.Do(func() {
		if r.cancel != nil {
			r.cancel()
		}
		if r.release != nil {
			r.release()
		}
	})
}
