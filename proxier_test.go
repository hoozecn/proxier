package proxier

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/integration"
)

func Test_parseConfig(t *testing.T) {
	assert := require.New(t)
	os.Setenv("PROXIER_ETCD_ENDPOINTS", "123")
	os.Setenv("PROXIER_SELF_IP", "123")
	cfg, err := parseSvcConfig()
	assert.Nil(err)
	assert.Equal("123", cfg.Endpoints[0])
	assert.Equal(int16(1080), cfg.SocksPort)
	assert.Equal(int16(1087), cfg.HttpPort)
}

func Test_register(t *testing.T) {
	cfg := integration.ClusterConfig{Size: 1}
	cluster := integration.NewClusterV3(t, &cfg)
	defer cluster.Terminate(t)

	os.Setenv("PROXIER_ETCD_ENDPOINTS", cluster.Client(0).Endpoints()[0])
	os.Setenv("PROXIER_SELF_IP", "127.0.0.1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var err error
	go func() {
		err = RegisterAndServe(ctx)
		require.New(t).Nil(err)
	}()

	<-time.After(2 * time.Second)
	nodes, err := List(ctx)
	require.New(t).Nil(err)
	require.New(t).Len(nodes, 1)
}
