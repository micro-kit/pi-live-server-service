package test

import (
	"context"
	"encoding/json"
	"log"
	"testing"
	"time"

	"github.com/micro-kit/microkit-client/client/piliveserver"
	"github.com/micro-kit/microkit-client/proto/piliveserverpb"
)

var (
	cl piliveserverpb.PiLiveServerClient
)

func TestMain(m *testing.M) {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	var err error
	cl, err = piliveserver.NewClient()
	if err != nil {
		log.Panicln(err)
	}
	m.Run()
}

func TestGetLiveUrl(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	info, err := cl.GetLiveUrl(ctx, &piliveserverpb.GetLiveUrlRequest{
		LiveId: "99",
	})
	if err != nil {
		log.Println(err)
		return
	}
	js, _ := json.Marshal(info)
	log.Println(string(js))
}

func TestQueryLivesByApp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	lives, err := cl.QueryLivesByApp(ctx, &piliveserverpb.QueryLivesByAppRequest{
		Appname: "live",
	})
	if err != nil {
		log.Println(err)
		return
	}
	js, _ := json.Marshal(lives)
	log.Println(string(js))
}
