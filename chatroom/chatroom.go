package chatroom

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
)

type chatroom struct {
	srv *http.Server
	m   *melody.Melody
	id  int
	mux *sync.Mutex
}

func New(port int) *chatroom {
	c := &chatroom{
		id:  0,
		mux: &sync.Mutex{},
	}

	c.setMelody()
	c.setServer(port)
	return c
}

func (c *chatroom) setMelody() {
	c.m = melody.New()

	c.m.HandleConnect(func(s *melody.Session) {
		c.mux.Lock()
		defer c.mux.Unlock()
		c.id++
		s.Set("id", c.id)
		c.m.Broadcast([]byte(fmt.Sprintf("|%s| <大廳> id %d join", getNowTimeStr(), c.id)))
	})

	c.m.HandleMessage(func(s *melody.Session, b []byte) {
		id, ok := s.Get("id")
		if !ok {
			return
		}
		c.m.Broadcast([]byte(fmt.Sprintf("|%s| <%v> %s", getNowTimeStr(), id, b)))
	})

	c.m.HandleClose(func(s1 *melody.Session, i int, s2 string) error {
		id, ok := s1.Get("id")
		if !ok {
			return nil
		}
		c.m.Broadcast([]byte(fmt.Sprintf("|%s| <大廳> id %d bye!", getNowTimeStr(), id)))
		return nil
	})
}

func (c *chatroom) setServer(port int) {
	route := gin.Default()
	route.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})

	route.GET("/ws", func(ctx *gin.Context) {
		c.m.HandleRequest(ctx.Writer, ctx.Request)
	})

	c.srv = &http.Server{
		Handler: route,
		Addr:    fmt.Sprintf(":%d", port),
	}
}

func (c *chatroom) Run() {
	go func() {
		if err := c.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Panicln("listen err:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown Server ...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
	defer cancel()
	c.m.Close()
	if err := c.srv.Shutdown(ctx); err != nil {
		log.Panicln("server shutdown err:", err)
	}
	log.Println("Server exited")
}

func getNowTimeStr() string {
	return time.Now().Format(time.ANSIC)
}
