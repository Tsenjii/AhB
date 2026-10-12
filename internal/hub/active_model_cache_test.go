package hub

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Tsenjii/AhB/internal/config"
)

func TestActiveModelsSingleflightBoundedAndFailure(t *testing.T) {
	var requests atomic.Int32
	var fail atomic.Bool
	srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		requests.Add(1)
		time.Sleep(30*time.Millisecond)
		if fail.Load(){w.WriteHeader(503);return}
		w.Header().Set("Content-Type","application/json")
		fmt.Fprint(w,`{"data":[{"id":"alpha"}]}`)
	}))
	defer srv.Close()
	h:=New(config.Config{})
	cfg:=config.ProviderConfig{ID:"test",DisplayName:"Local",BaseURL:srv.URL,ModelsPath:"/v1/models"}
	var wg sync.WaitGroup
	failures:=make(chan error,40)
	for i:=0;i<40;i++ {
		wg.Add(1)
		go func(){
			defer wg.Done()
			models,err:=h.fetchActiveModels(context.Background(),cfg)
			if err!=nil {failures<-err;return}
			if len(models)!=1 || models[0]["id"]!="test/alpha" {
				failures<-fmt.Errorf("bad catalog %v",models)
			}
		}()
	}
	wg.Wait();close(failures)
	for err:=range failures {t.Error(err)}
	if count:=requests.Load();count!=1 {
		t.Fatalf("concurrent calls should probe upstream once, got %d",count)
	}
	models,err:=h.fetchActiveModels(context.Background(),cfg)
	if err!=nil{t.Fatal(err)}
	models[0]["id"]="tampered"
	again,err:=h.fetchActiveModels(context.Background(),cfg)
	if err!=nil || again[0]["id"]!="test/alpha" {t.Fatal("cache mutation leaked across requests")}
	// Force expiry without waiting fifteen real seconds.
	h.activeModelCache.mu.Lock()
	h.activeModelCache.entries[cfg.ID].updated=time.Now().Add(-time.Minute)
	h.activeModelCache.mu.Unlock()
	fail.Store(true)
	if _,err=h.fetchActiveModels(context.Background(),cfg);err==nil {t.Fatal("fresh 503 treated as success")}
	fail.Store(false)
	if _,err=h.fetchActiveModels(context.Background(),cfg);err!=nil {t.Fatalf("recovery failed: %v",err)}
	if requests.Load()!=3 {t.Fatalf("expected one probe, one error, one recovery: got %d",requests.Load())}
}

func TestActiveModelsWaitingCanCancel(t *testing.T) {
	started:=make(chan struct{})
	block:=make(chan struct{})
	srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		close(started)
		<-block
		fmt.Fprint(w,`{"data":[]}`)
	}))
	defer srv.Close()
	h:=New(config.Config{})
	cfg:=config.ProviderConfig{ID:"slow",BaseURL:srv.URL,ModelsPath:"/v1/models"}
	result:=make(chan error,1)
	go func(){_,err:=h.fetchActiveModels(context.Background(),cfg);result<-err}()
	<-started
	ctx,cancel:=context.WithCancel(context.Background());cancel()
	_,err:=h.fetchActiveModels(ctx,cfg)
	if err==nil {t.Fatal("cancelled waiter hung or accepted stale data")}
	close(block)
	if err:=<-result;err!=nil {t.Fatal(err)}
}
