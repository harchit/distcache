package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/memberlist"
)

type DistributedCache struct {
	cache  *Cache
	list   *memberlist.Memberlist
	config *memberlist.Config
}

func (dc *DistributedCache) joinCluster(peer string) error {
	_, err := dc.list.Join([]string{peer})
	return err
}

func (dc *DistributedCache) httpHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/cache/"):]
	switch r.Method {
	case http.MethodGet:
		value, found := dc.cache.Get(key)
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, "%v", value)
	case http.MethodPost:
		value := r.PostFormValue("value")
		durationStr := r.PostFormValue("duration")
		duration, err := strconv.ParseInt(durationStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		dc.cache.Set(key, value, time.Duration(duration))
	case http.MethodDelete:
		dc.cache.Delete(key)
	}
}

func newDistributedCache(port int) (*DistributedCache, error) {
	cache := NewCache()
	config := memberlist.DefaultLANConfig()
	config.BindPort = port
	config.AdvertisePort = port
	list, err := memberlist.Create(config)
	if err != nil {
		return nil, err
	}
	dc := &DistributedCache{
		cache:  cache,
		list:   list,
		config: config,
	}
	return dc, nil

}

func main() {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	peer := os.Getenv("PEER")
	dc, err := newDistributedCache(port)
	if err != nil {
		log.Fatalf("Failed to create distributed cache: %v", err)
	}
	if peer != "" {
		err = dc.joinCluster(peer)
		if err != nil {
			log.Fatalf("Failed to join cluster: %v", err)
		}
	}
	http.HandleFunc("/cache/", dc.httpHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))

}
