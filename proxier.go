package proxier

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/coreos/etcd/clientv3"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

const REGISTRY_PREFIX = "/registry/proxier"

type svcConfig struct {
	Endpoints []string `envconfig:"PROXIER_ETCD_ENDPOINTS" required:"true"`
	SelfIP    string   `envconfig:"PROXIER_SELF_IP" required:"true"`
	HttpPort  int16    `envconfig:"PROXIER_HTTP_PORT" default:"1087"`
	SocksPort int16    `envconfig:"PROXIER_SOCKS_PORT" default:"1080"`
	username  string
	password  string
	uuid      string
}

func (c *svcConfig) getNode() *Node {
	return &Node{
		HTTP:  fmt.Sprintf("%s:%d", c.SelfIP, c.HttpPort),
		Socks: fmt.Sprintf("%s:%d", c.SelfIP, c.SocksPort),
		IP:    c.SelfIP,
	}
}

type Node struct {
	HTTP  string `json:"http"`
	Socks string `json:"socks"`
	IP    string `json:"ip"`
}

func parseSvcConfig() (*svcConfig, error) {
	var cfg svcConfig
	err := envconfig.Process("", &cfg)
	return &cfg, err
}

func RegisterAndServe(ctx context.Context) (err error) {
	var (
		cli       *clientv3.Client
		cfg       *svcConfig
		leaseResp *clientv3.LeaseGrantResponse
		svcDone   chan error
	)

	svcDone = make(chan error, 1)
	if cfg, err = parseSvcConfig(); err != nil {
		return err
	}

	go func() {
		svcErr := serve(ctx, cfg)
		svcDone <- svcErr
	}()

	if cli, err = clientv3.New(clientv3.Config{
		Endpoints: cfg.Endpoints,
	}); err != nil {
		return err
	}

	svcCtx := context.Background()

	if leaseResp, err = cli.Grant(ctx, 10); err != nil {
		return err
	}

	node, _ := json.Marshal(cfg.getNode())
	if _, err = cli.Put(svcCtx, REGISTRY_PREFIX+"/"+uuid.NewString(), string(node), clientv3.WithLease(leaseResp.ID)); err != nil {
		return err
	}

	defer func() {
		_, err = cli.Revoke(svcCtx, leaseResp.ID)
	}()

	var respChan <-chan *clientv3.LeaseKeepAliveResponse
	if respChan, err = cli.KeepAlive(ctx, leaseResp.ID); err != nil {
		return err
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	for {
		select {
		case <-ctx.Done():
			return
		case err = <-svcDone:
			if err != nil {
				logger.Info("svc done")
			}
		case <-respChan:
		}
	}
}

func serve(ctx context.Context, cfg *svcConfig) (err error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	httpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
	if err != nil {
		return err
	}

	socksListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.SocksPort))
	if err != nil {
		return err
	}

	go serveHttp(ctx, httpListener, logger)
	go serveSocks(ctx, socksListener, logger)

	<-ctx.Done()
	httpListener.Close()
	socksListener.Close()
	return nil
}

func serveSocks(ctx context.Context, l net.Listener, logger logrus.FieldLogger) error {
	logger.Infof("socks proxy started at %s", l.Addr())
	for {
		client, err := l.Accept()
		if err != nil {
			return err
		}

		go processS5(client, logger)
	}
}
