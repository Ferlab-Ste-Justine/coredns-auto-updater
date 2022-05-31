package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdClient struct {
	Client *clientv3.Client
	Retries uint64
	RequestTimeout uint64
}

func Connect(
	userCertPath string, 
	userKeyPath string,
	Username string,
	Password string, 
	caCertPath string, 
	etcdEndpoints string, 
	connectionTimeout uint64,
	requestTimeout uint64,
	retries uint64,
	) (*EtcdClient, error) {
	tlsConf := &tls.Config{}

	//User credentials
	if Username == "" {
		certData, err := tls.LoadX509KeyPair(userCertPath, userKeyPath)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to load user credentials: %s", err.Error()))
		}
		(*tlsConf).Certificates = []tls.Certificate{certData}
	}

	(*tlsConf).InsecureSkipVerify = false
	
	//CA cert
	caCertContent, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to read root certificate file: %s", err.Error()))
	}
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(caCertContent)
	if !ok {
		return nil, errors.New("Failed to parse root certificate authority")
	}
	(*tlsConf).RootCAs = roots
	
	//Connection
	var cli *clientv3.Client
	var connErr error

	if Username == "" {
		cli, connErr = clientv3.New(clientv3.Config{
			Endpoints:   strings.Split(etcdEndpoints, ","),
			TLS:         tlsConf,
			DialTimeout: time.Duration(connectionTimeout) * time.Second,
		})
	} else {
		cli, connErr = clientv3.New(clientv3.Config{
			Username: Username,
			Password: Password,
			Endpoints:   strings.Split(etcdEndpoints, ","),
			TLS:         tlsConf,
			DialTimeout: time.Duration(connectionTimeout) * time.Second,
		})
	}
	
	if connErr != nil {
		return nil, errors.New(fmt.Sprintf("Failed to connect to etcd servers: %s", connErr.Error()))
	}
	
	return &EtcdClient{
		Client:         cli,
		Retries:        retries,
		RequestTimeout: requestTimeout,
	}, nil
}

func (cli *EtcdClient) getZonefilesRecursive(etcdKeyPrefix string, retries uint64) (map[string]string, int64, error) {
	var cancel context.CancelFunc
	ctx := context.Background()
	zonefiles := make(map[string]string)
	if cli.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(cli.RequestTimeout) * time.Second)
		defer cancel()
	}
	res, err := cli.Client.Get(ctx, etcdKeyPrefix, clientv3.WithPrefix())
	if err != nil {
		etcdErr, ok := err.(rpctypes.EtcdError)
		if !ok {
			return zonefiles, 0, errors.New(fmt.Sprintf("Failed to retrieve zonefiles: %s", err.Error()))
		}
		
		if etcdErr.Code() != codes.Unavailable || retries == 0 {
			return zonefiles, 0, errors.New(fmt.Sprintf("Failed to retrieve zonefiles: %s", etcdErr.Error()))
		}

		time.Sleep(time.Duration(100) * time.Millisecond)
		return cli.getZonefilesRecursive(etcdKeyPrefix, retries - 1)
	}

	for _, kv := range res.Kvs {
		zonefiles[strings.TrimPrefix(string(kv.Key), etcdKeyPrefix)] = string(kv.Value)
	}
	
	return zonefiles, res.Header.Revision, nil
}

func (cli *EtcdClient) GetZonefiles(etcdKeyPrefix string) (map[string]string, int64, error) {
	return cli.getZonefilesRecursive(etcdKeyPrefix, cli.Retries)
}

type ZonefileEvent struct {
	Domain  string
	Content string
	Action  string
	Err     error
}

func (cli *EtcdClient) WatchZonefiles(etcdKeyPrefix string, revision int64, events chan ZonefileEvent) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer close(events)
	wc := cli.Client.Watch(ctx, etcdKeyPrefix, clientv3.WithPrefix(), clientv3.WithRev(revision))
	if wc == nil {
		events <- ZonefileEvent{Err: errors.New("Failed to watch zonefiles changes: Watcher could not be established")}
		return
	}

	for res := range wc {
		err := res.Err()
		if err != nil {
			events <- ZonefileEvent{Err: errors.New(fmt.Sprintf("Failed to watch zonefiles changes: %s", err.Error()))}
			return
		}

		for _, ev := range res.Events {
			if ev.Type == mvccpb.DELETE {
				events <- ZonefileEvent{Action: "delete", Domain: strings.TrimPrefix(string(ev.Kv.Key), etcdKeyPrefix)}
			} else if ev.Type == mvccpb.PUT {
				events <- ZonefileEvent{Action: "upsert", Domain: strings.TrimPrefix(string(ev.Kv.Key), etcdKeyPrefix), Content: string(ev.Kv.Value)}
			}
		}
	}
}