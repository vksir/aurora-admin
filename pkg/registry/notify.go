package registry

import (
	"context"
	"aurora-admin/pkg/log"
	"sync"
)

var gTopics = make(map[string]map[string]Callback)
var gTopicsLock sync.RWMutex

type Callback = func(ctx context.Context, msgAny any) error

func Notify(ctx context.Context, topic string, msg any) {
	gTopicsLock.RLock()
	defer gTopicsLock.RUnlock()

	log.InfoC(ctx, "notify", "topic", topic, "msg", msg)
	tp, ok := gTopics[topic]
	if ok {
		for subs, callback := range tp {
			err := callback(ctx, msg)
			if err != nil {
				log.ErrorC(ctx, "callback failed", "topic", topic, "subs", subs, "msg", msg)
			}
		}
	}
}

func Subscribe(topic, subscriber string, callback Callback) {
	gTopicsLock.Lock()
	defer gTopicsLock.Unlock()

	tp, ok := gTopics[topic]
	if !ok {
		tp = make(map[string]Callback)
		gTopics[topic] = tp
	}
	tp[subscriber] = callback
	log.Warn("subscribe success", "topic", topic, "subscriber", subscriber, "callback", callback)
}

func Unsubscribe(topic, subscriber string) {
	gTopicsLock.Lock()
	defer gTopicsLock.Unlock()

	tp, ok := gTopics[topic]
	if ok {
		delete(tp, subscriber)
		if len(tp) == 0 {
			delete(gTopics, topic)
		}
	}
	log.Warn("unsubscribe success", "topic", topic, "subscriber", subscriber)
}
