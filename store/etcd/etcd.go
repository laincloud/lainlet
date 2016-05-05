package etcd

import (
	"crypto/tls"
	etcd "github.com/coreos/etcd/client"
	"github.com/laincloud/lainlet/store"
	"golang.org/x/net/context"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Etcd is the receiver type for the
// Store interface
type Etcd struct {
	client etcd.KeysAPI
}

var (
	actionMap = map[string]store.Action{
		"get":              store.GET,
		"compareAndSwap":   store.UPDATE,
		"set":              store.UPDATE,
		"update":           store.UPDATE,
		"create":           store.UPDATE,
		"delete":           store.DELETE,
		"error":            store.ERROR,
		"expire":           store.DELETE,
		"compareAndDelete": store.DELETE,
	}
)

func init() {
	store.Register("etcd", New)
}

// New creates a new Etcd client given a list
// of endpoints and an optional tls config
func New(addrs []string) (store.Store, error) {
	s := &Etcd{}

	var (
		entries []string
		err     error
	)

	entries = store.CreateEndpoints(addrs, "http")
	cfg := &etcd.Config{
		Endpoints:               entries,
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: 3 * time.Second,
	}

	c, err := etcd.New(*cfg)
	if err != nil {
		log.Fatal(err)
	}

	s.client = etcd.NewKeysAPI(c)

	return s, nil
}

// SetTLS sets the tls configuration given a tls.Config scheme
func setTLS(cfg *etcd.Config, tls *tls.Config, addrs []string) {
	entries := store.CreateEndpoints(addrs, "https")
	cfg.Endpoints = entries

	// Set transport
	t := http.Transport{
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tls,
	}

	cfg.Transport = &t
}

// Normalize the key for usage in Etcd
func (s *Etcd) normalize(key string) string {
	key = store.Normalize(key)
	return strings.TrimPrefix(key, "/")
}

// keyNotFound checks on the error returned by the KeysAPI
// to verify if the key exists in the store or not
func keyNotFound(err error) bool {
	if err != nil {
		if etcdError, ok := err.(etcd.Error); ok {
			if etcdError.Code == etcd.ErrorCodeKeyNotFound ||
				etcdError.Code == etcd.ErrorCodeNotFile ||
				etcdError.Code == etcd.ErrorCodeNotDir {
				return true
			}
		}
	}
	return false
}

func deleteEvent(key string) *store.Event {
	return &store.Event{
		Action: actionMap["delete"],
		Key:    key,
		Data:   []*store.KVPair{},
	}
}

func errorEvent(key string, err error) *store.Event {
	return &store.Event{
		Action: store.ERROR,
		Key:    key,
		Data:   []*store.KVPair{&store.KVPair{"error", []byte(err.Error()), 0}},
	}
}

// Get the value, returns the last modified
func (s *Etcd) Get(key string) (pair *store.KVPair, err error) {
	getOpts := &etcd.GetOptions{
		Quorum: true,
	}

	result, err := s.client.Get(context.Background(), s.normalize(key), getOpts)
	if err != nil {
		if keyNotFound(err) {
			return nil, store.ErrKeyNotFound
		}
		return nil, err
	}

	pair = &store.KVPair{
		Key:       key,
		Value:     []byte(result.Node.Value),
		LastIndex: result.Node.ModifiedIndex,
	}

	return pair, nil
}

// List all the node's name in a directory
func (s *Etcd) List(dir string) ([]string, error) {
	getOpts := &etcd.GetOptions{
		Quorum: true,
	}

	result, err := s.client.Get(context.Background(), s.normalize(dir), getOpts)
	if err != nil {
		if keyNotFound(err) {
			return nil, store.ErrKeyNotFound
		}
		return nil, err
	}

	var nodes []string

	for _, node := range result.Node.Nodes {
		nodes = append(nodes, node.Key)
	}

	return nodes, nil
}

// Watch for changes on a "key"
// It returns a channel that will receive changes or pass
// on errors. Upon creation, the current value will first
// be sent to the channel. Providing a non-nil stopCh can
// be used to stop watching.
func (s *Etcd) Watch(key string, ctx context.Context, recursive bool, index uint64) (<-chan *store.Event, error) {
	opts := &etcd.WatcherOptions{Recursive: recursive, AfterIndex: index}
	watcher := s.client.Watcher(s.normalize(key), opts)

	// watchCh is sending back events to the caller
	watchCh := make(chan *store.Event)

	go func() {
		defer close(watchCh)
		for {
			result, err := watcher.Next(ctx)

			if err != nil {
				// Push error value through the channel.
				watchCh <- errorEvent(key, err)
				return
			}
			watchCh <- &store.Event{
				Action:        actionMap[result.Action],
				Key:           result.Node.Key,
				ModifiedIndex: result.Node.ModifiedIndex,
				Data: []*store.KVPair{
					&store.KVPair{
						Key:       result.Node.Key,
						Value:     []byte(result.Node.Value),
						LastIndex: result.Node.ModifiedIndex,
					},
				},
			}
		}
	}()

	return watchCh, nil
}

// WatchTree watches for changes on a "directory"
// It returns a channel that will receive changes or pass
// on errors. Upon creating a watch, the current childs values
// will be sent to the channel.
func (s *Etcd) WatchTree(directory string, ctx context.Context, index uint64) (<-chan *store.Event, error) {
	watchOpts := &etcd.WatcherOptions{Recursive: true, AfterIndex: index}
	watcher := s.client.Watcher(s.normalize(directory), watchOpts)

	// watchCh is sending back events to the caller
	watchCh := make(chan *store.Event, 1)

	go func() {
		defer close(watchCh)
		for {
			resp, err := watcher.Next(ctx)

			if err != nil {
				watchCh <- errorEvent(directory, err)
				return
			}

			// direcotry was delete, send a action, stop to watch
			if resp.Action == "delete" && resp.Node.Key == directory {
				watchCh <- deleteEvent(resp.Node.Key)
				return
			}

			data, err := s.GetTree(directory)
			if err != nil {
				if err == store.ErrKeyNotFound {
					// key not found, means it was deleted before s.GetTree()
					// FIXME: it's ok? may lost some event if changed too fast, use modifiedIndex to GetTree()?
					watchCh <- deleteEvent(directory)
					continue
				} else {
					watchCh <- errorEvent(directory, err)
					return
				}
			}

			watchCh <- &store.Event{
				Action:        actionMap[resp.Action],
				Key:           resp.Node.Key,
				ModifiedIndex: resp.Node.ModifiedIndex,
				Data:          data,
			}
		}
	}()

	return watchCh, nil
}

// GetTree get child nodes of a given directory
func (s *Etcd) GetTree(directory string) ([]*store.KVPair, error) {
	getOpts := &etcd.GetOptions{
		Quorum:    true,
		Recursive: true,
		Sort:      true,
	}

	resp, err := s.client.Get(context.Background(), s.normalize(directory), getOpts)
	if err != nil {
		if keyNotFound(err) {
			return nil, store.ErrKeyNotFound
		}
		return nil, err
	}
	if !resp.Node.Dir { // it is a key, not a directory
		return []*store.KVPair{
			&store.KVPair{
				Key:       resp.Node.Key,
				Value:     []byte(resp.Node.Value),
				LastIndex: resp.Node.ModifiedIndex,
			},
		}, nil
	}

	return travelNodes(resp.Node.Nodes), nil
}

// Put a value
func (s *Etcd) Put(key string, value []byte) error {
	setOpts := &etcd.SetOptions{}
	_, err := s.client.Set(context.Background(), s.normalize(key), string(value), setOpts)
	return err
}

// Delete a value by given key
func (s *Etcd) Delete(key string, recursive bool) error {
	opts := &etcd.DeleteOptions{
		Recursive: recursive,
	}

	_, err := s.client.Delete(context.Background(), s.normalize(key), opts)
	if keyNotFound(err) {
		return store.ErrKeyNotFound
	}
	return err
}

// Close closes the client connection
func (s *Etcd) Close() {
	return
}

func travelNodes(nodes []*etcd.Node) []*store.KVPair {
	var pairs []*store.KVPair
	for _, node := range nodes {
		if node.Dir {
			pairs = append(pairs, travelNodes(node.Nodes)...)
		} else {
			pairs = append(pairs, &store.KVPair{
				Key:       node.Key,
				Value:     []byte(node.Value),
				LastIndex: node.ModifiedIndex,
			})
		}
	}
	return pairs
}

// Exists checks if the key exists inside the store
func (s *Etcd) Exists(key string) (bool, error) {
	_, err := s.Get(key)
	if err != nil {
		if err == store.ErrKeyNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
