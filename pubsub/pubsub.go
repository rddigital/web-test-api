// package pubsub implements a simple multi-topic pub-sub library.
package pubsub

import (
	"sync"
	"time"
)

type (
	// subscriber thuộc kiểu channel
	subscriber chan interface{}

	// topic là một filter
	topicFunc func(v interface{}) bool
)

type Publisher struct {
	// Read/Write Mutex
	m sync.RWMutex

	// kích thước  hàng đợi
	buffer int

	// timeout cho việc publishing
	timeout time.Duration

	// subscriber đã subscribe vào topic nào
	subscribers map[subscriber]topicFunc
}

// constructor với timeout và độ dài hàng đợi
func NewPublisher(publishTimeout time.Duration, buffer int) *Publisher {
	return &Publisher{
		buffer:      buffer,
		timeout:     publishTimeout,
		subscribers: make(map[subscriber]topicFunc),
	}
}

// thêm subscriber mới, đăng ký hết tất cả topic
func (p *Publisher) Subscribe() chan interface{} {
	return p.SubscribeTopic(nil)
}

// thêm subscriber mới, subscribe các topic đã được filter lọc
func (p *Publisher) SubscribeTopic(topic topicFunc) chan interface{} {
	ch := make(chan interface{}, p.buffer)
	p.m.Lock()
	p.subscribers[ch] = topic
	p.m.Unlock()
	return ch
}

// hủy subscribe
func (p *Publisher) Evict(sub chan interface{}) {
	p.m.Lock()
	defer p.m.Unlock()

	delete(p.subscribers, sub)
	close(sub)
}

// publish ra 1 topic
func (p *Publisher) Publish(v interface{}) {
	p.m.RLock()
	defer p.m.RUnlock()

	var wg sync.WaitGroup
	for sub, topic := range p.subscribers {
		wg.Add(1)
		go p.sendTopic(sub, topic, v, &wg)
	}
	wg.Wait()
}

// đóng 1 đối tượng publisher và đóng tất cả các subscriber
func (p *Publisher) Close() {
	p.m.Lock()
	defer p.m.Unlock()

	for sub := range p.subscribers {
		close(sub)
		delete(p.subscribers, sub)
	}
}

// gửi 1 topic có thể duy trì trong thời gian chờ wg
func (p *Publisher) sendTopic(sub subscriber, topic topicFunc, v interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	if topic != nil && !topic(v) {
		return
	}

	select {
	case sub <- v:
	case <-time.After(p.timeout):
	}
}
