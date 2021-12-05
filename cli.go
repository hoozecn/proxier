package proxier

import (
	"context"
	"encoding/json"

	"github.com/kelseyhightower/envconfig"
	"go.etcd.io/etcd/clientv3"
)

type clientConfig struct {
	Endpoints []string `envconfig:"PROXIER_ETCD_ENDPOINTS" required:"true"`
}

func List(ctx context.Context) (nodeList []*Node, err error) {
	var cfg *clientConfig
	if cfg, err = parseCliConfig(); err != nil {
		return
	}

	var cli *clientv3.Client
	if cli, err = clientv3.New(clientv3.Config{Endpoints: cfg.Endpoints}); err != nil {
		return
	}

	var resp *clientv3.GetResponse
	if resp, err = cli.Get(ctx, REGISTRY_PREFIX, clientv3.WithPrefix()); err != nil {
		return
	}

	for _, kv := range resp.Kvs {
		var node *Node
		err = json.Unmarshal(kv.Value, &node)
		if err != nil {
			return
		}

		nodeList = append(nodeList, node)
	}

	return
}

func parseCliConfig() (*clientConfig, error) {
	var cfg clientConfig
	err := envconfig.Process("", &cfg)
	return &cfg, err
}
