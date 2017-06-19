package targetcache

import (
	"bytes"
	"encoding/json"
	"sync"

	"go.uber.org/zap"

	"github.com/serverless/gateway/endpoints"
	"github.com/serverless/gateway/functions"
	"github.com/serverless/gateway/pubsub"
)

// Cache is a simplification of the db.Reactive interface, which doesn't care about
// the distinction between Created and Modified, reducing them to Set.
type Cache interface {
	Set(string, []byte)
	Del(string, []byte)
}

type cacheMaintainer struct {
	cache Cache
}

func newCacheMaintainer(cache Cache) *cacheMaintainer {
	return &cacheMaintainer{
		cache: cache,
	}
}

// Created is called when a new endpoint is detected in the config.
func (c *cacheMaintainer) Created(key string, value []byte) {
	c.cache.Set(key, value)
}

// Modified is called when an existing endpoint is modified in the config.
func (c *cacheMaintainer) Modified(key string, newValue []byte) {
	c.cache.Set(key, newValue)
}

// Deleted is called when a endpoint is deleted in the config.
func (c *cacheMaintainer) Deleted(key string, lastKnownValue []byte) {
	c.cache.Del(key, lastKnownValue)
}

type functionCache struct {
	sync.RWMutex
	// cache maps from FunctionID to Function
	cache map[functions.FunctionID]functions.Function
	log   *zap.Logger
}

func newFunctionCache(log *zap.Logger) *functionCache {
	return &functionCache{
		cache: map[functions.FunctionID]functions.Function{},
		log:   log,
	}
}

func (c *functionCache) Set(k string, v []byte) {
	f := functions.Function{}
	err := json.NewDecoder(bytes.NewReader(v)).Decode(&f)
	if err != nil {
		c.log.Error("Could not deserialize Function state!", zap.Error(err), zap.String("key", k))
	} else {
		c.Lock()
		defer c.Unlock()
		c.cache[functions.FunctionID(k)] = f
	}
}

func (c *functionCache) Del(k string, v []byte) {
	c.Lock()
	defer c.Unlock()
	delete(c.cache, functions.FunctionID(k))
}

type endpointCache struct {
	sync.RWMutex
	// cache maps from EndpointID to Endpoint
	cache map[endpoints.EndpointID]endpoints.Endpoint
	log   *zap.Logger
}

func newEndpointCache(log *zap.Logger) *endpointCache {
	return &endpointCache{
		cache: map[endpoints.EndpointID]endpoints.Endpoint{},
		log:   log,
	}
}

func (c *endpointCache) Set(k string, v []byte) {
	e := endpoints.Endpoint{}
	err := json.NewDecoder(bytes.NewReader(v)).Decode(&e)
	if err != nil {
		c.log.Error("Could not deserialize Endpoint state!", zap.Error(err), zap.String("key", k))
	} else {
		c.Lock()
		defer c.Unlock()
		c.cache[endpoints.EndpointID(k)] = e
	}
}

func (c *endpointCache) Del(k string, v []byte) {
	c.Lock()
	defer c.Unlock()
	delete(c.cache, endpoints.EndpointID(k))
}

type publisherCache struct {
	sync.RWMutex

	// cache keeps deserialized Publishers around to properly delete them later
	cache map[pubsub.PublisherID]pubsub.Publisher

	// fnInToTopic maps from FunctionID to a set of TopicID's that consume the input of the function
	fnInToTopic map[functions.FunctionID]map[pubsub.TopicID]struct{}

	// fnOutToTopic maps from FunctionID to a set of TopicID's that consume the output of the function
	fnOutToTopic map[functions.FunctionID]map[pubsub.TopicID]struct{}

	log *zap.Logger
}

func newPublisherCache(log *zap.Logger) *publisherCache {
	return &publisherCache{
		log:          log,
		fnInToTopic:  map[functions.FunctionID]map[pubsub.TopicID]struct{}{},
		fnOutToTopic: map[functions.FunctionID]map[pubsub.TopicID]struct{}{},
	}
}

func (c *publisherCache) Set(k string, v []byte) {
	p := pubsub.Publisher{}
	err := json.NewDecoder(bytes.NewReader(v)).Decode(&p)
	if err != nil {
		c.log.Error("Could not deserialize Publisher state!", zap.Error(err), zap.String("key", k))
		return
	}

	c.Lock()
	defer c.Unlock()

	c.cache[pubsub.PublisherID(k)] = p

	if p.FunctionEnd == pubsub.Input {
		pubSet, exists := c.fnInToTopic[p.FunctionID]
		if exists {
			pubSet[p.TopicID] = struct{}{}
		} else {
			pubSet := map[pubsub.TopicID]struct{}{}
			pubSet[p.TopicID] = struct{}{}
			c.fnInToTopic[p.FunctionID] = pubSet
		}
	} else if p.FunctionEnd == pubsub.Output {
		pubSet, exists := c.fnOutToTopic[p.FunctionID]
		if exists {
			pubSet[p.TopicID] = struct{}{}
		} else {
			pubSet := map[pubsub.TopicID]struct{}{}
			pubSet[p.TopicID] = struct{}{}
			c.fnOutToTopic[p.FunctionID] = pubSet
		}
	} else {
		c.log.Error("received a new Publisher with an invalid FunctionEnd!")
	}
}

func (c *publisherCache) Del(k string, v []byte) {
	c.Lock()
	defer c.Unlock()

	oldProd, exists := c.cache[pubsub.PublisherID(k)]
	if !exists {
		return
	}

	if oldProd.FunctionEnd == pubsub.Input {
		inTopicSet, exists := c.fnInToTopic[oldProd.FunctionID]
		if exists {
			delete(inTopicSet, oldProd.TopicID)

			if len(inTopicSet) == 0 {
				delete(c.fnInToTopic, oldProd.FunctionID)
			}
		}
	} else if oldProd.FunctionEnd == pubsub.Output {
		outTopicSet, exists := c.fnOutToTopic[oldProd.FunctionID]
		if exists {
			delete(outTopicSet, oldProd.TopicID)

			if len(outTopicSet) == 0 {
				delete(c.fnOutToTopic, oldProd.FunctionID)
			}
		}
	} else {
		c.log.Error("trying to delete a Publisher with an invalid FunctionEnd!")
	}
}

type subscriberCache struct {
	sync.RWMutex
	// topicToSub maps from a TopicID to a set of subscribing FunctionID's
	topicToFns map[pubsub.TopicID]map[functions.FunctionID]struct{}
	log        *zap.Logger
}

func newSubscriberCache(log *zap.Logger) *subscriberCache {
	return &subscriberCache{
		// topicToFns is a map from TopicID to a set of FunctionID's
		topicToFns: map[pubsub.TopicID]map[functions.FunctionID]struct{}{},
		log:        log,
	}
}

func (c *subscriberCache) Set(k string, v []byte) {
	s := pubsub.Subscriber{}
	err := json.NewDecoder(bytes.NewReader(v)).Decode(&s)
	if err != nil {
		c.log.Error("Could not deserialize Subscriber state!", zap.Error(err), zap.String("key", k))
		return
	}

	c.Lock()
	defer c.Unlock()

	// set FunctionID as destination in topicToSub
	fnSet, exists := c.topicToFns[s.TopicID]
	if exists {
		fnSet[s.FunctionID] = struct{}{}
	} else {
		fnSet := map[functions.FunctionID]struct{}{}
		fnSet[s.FunctionID] = struct{}{}
		c.topicToFns[s.TopicID] = fnSet
	}
}

func (c *subscriberCache) Del(k string, v []byte) {
	c.Lock()
	defer c.Unlock()

	oldSub := pubsub.Subscriber{}
	err := json.NewDecoder(bytes.NewReader(v)).Decode(&oldSub)
	if err != nil {
		c.log.Error("Could not deserialize Subscriber state during deletion!", zap.Error(err), zap.String("key", k))
		return
	}

	fnSet, exists := c.topicToFns[oldSub.TopicID]
	if exists {
		delete(fnSet, oldSub.FunctionID)

		if len(fnSet) == 0 {
			delete(c.topicToFns, oldSub.TopicID)
		}
	}
}