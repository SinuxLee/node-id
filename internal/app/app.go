package app

import (
	"net"
	"net/http"
	"nodeid/internal/config"
	"nodeid/internal/controller"
	"nodeid/internal/service"
	"nodeid/internal/store"
	"nodeid/pkg/nid"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// 常量定义
const (
	ServiceID = 110
)

// App ...
type App interface {
	Run(chan os.Signal) error
	Stop() error
}

// New ...
func New(options ...Option) (App, error) {
	svc := &app{}

	// init app component
	for _, opt := range options {
		if err := opt(svc); err != nil {
			return nil, err
		}
	}

	return svc, nil
}

type app struct {
	localIP string
	router  *gin.Engine
	httpSrv *http.Server
	conf    config.Conf
	ctrl    controller.Controller
	useCase service.UseCase
	dao     store.Dao
	named   nid.NodeNamed
}

func (s *app) GetServiceID() int {
	return ServiceID
}

func (s *app) GetLocalIP() string {
	if s.localIP == "" {
		s.localIP = s.intranetIP()
	}
	return s.localIP
}

// Run ...
func (s *app) Run(ch chan os.Signal) error {
	// Run server
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil {
			log.Error().Err(err).Msg("http app exit")
			close(ch)
		}
	}()

	return nil
}

// Stop ...
func (s *app) Stop() error {
	return nil
}

// intranetIP 找到第一个10、172、192开头的ip
func (s *app) intranetIP() (ip string) {
	addr, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	for _, addr := range addr {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	return
}
