package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/coreos/etcd/clientv3"
	"github.com/helmutkemper/communsTypesForGolangPlugin"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type Etcd struct {
	cli            *clientv3.Client
	hostList       string
	keyPrefix      string
	dialTimeOut    time.Duration
	requestTimeOut time.Duration
	onWatchFunc    func([]communsTypes.KeyValueType, []communsTypes.KeyValueType)
}

type PluginDataInterface interface {
	OnLoad(conf ...interface{}) error
	Connect() error
	Close() error
	Check() bool
	Put(value communsTypes.KeyValueType) error
	GetByPrefix(prefix []byte) (error, int, []communsTypes.KeyValueType)
	Get(key []byte) (error, int, []communsTypes.KeyValueType)
	SetOnWatch(watchFunc func([]communsTypes.KeyValueType, []communsTypes.KeyValueType))
	Watch(key []byte)
	Delete(key []byte) error
}

func (el *Etcd) handleError(err error) {
	if err != nil {
		_, fn, line, _ := runtime.Caller(1)
		log.Printf("[ETCD plugin error] %s:%d %v", fn, line, err)
	}
}

// on plugin load function
// conf[0] - string containing a json file path of configuration file
//
//   json example:
//   {
//     "hostList": "172.18.0.1:2379,172.18.0.2:2379,172.18.0.3:2379",
//     "keyPrefix": "dnsServerKey",
//     "dialTimeOut": 500000
//     "requestTimeOut": 1000000
//   }
func (el *Etcd) OnLoad(conf ...interface{}) error {
	var err error
	var fileContent []byte
	var jsonData map[string]interface{}
	var filePath = conf[0].([]interface{})[0].(string)

	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		el.handleError(err)
		return err
	}

	fileContent, err = ioutil.ReadFile(filePath)
	if err != nil {
		el.handleError(err)
		return err
	}

	err = json.Unmarshal(fileContent, &jsonData)
	if err != nil {
		el.handleError(err)
		return err
	}

	el.hostList = jsonData["hostList"].(string)
	if el.hostList == "" {
		err = errors.New("json hostList key not found")
		el.handleError(err)
		return err
	}

	el.keyPrefix = jsonData["keyPrefix"].(string)
	if el.keyPrefix == "" {
		err = errors.New("json keyPrefix key not found")
		el.handleError(err)
		return err
	}

	tm := jsonData["dialTimeOut"].(float64)
	if tm == 0 {
		err = errors.New("json dialTimeOut key not found")
		el.handleError(err)
		return err
	}
	el.dialTimeOut = time.Duration(tm) * time.Microsecond

	tm = jsonData["requestTimeOut"].(float64)
	if tm == 0 {
		err = errors.New("json requestTimeOut key not found")
		el.handleError(err)
		return err
	}
	el.requestTimeOut = time.Duration(tm) * time.Microsecond

	return nil
}

func (el *Etcd) Connect() error {
	var err error

	el.cli, err = clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(el.hostList, ","),
		DialTimeout: el.dialTimeOut,
	})

	return err
}

func (el *Etcd) Close() error {
	return el.cli.Close()
}

func (el *Etcd) Check() bool {
	/*
	  ctx, cancel := context.WithTimeout(context.Background(), el.requestTimeOut)
	  _, err := el.cli.Put(ctx, "test.etcd.server.connection", "bar", nil)
	  if err != nil {
	    if err == context.Canceled {
	      el.handleError( errors.New("Check() ctx is canceled by another routine. " + err.Error() ) )
	      return false
	    } else if err == context.DeadlineExceeded {
	      el.handleError( errors.New("Check() ctx is attached with a deadline and it exceeded. " + err.Error() ) )
	      return false
	    } else if cerr, ok := err.(*client.ClusterError); ok {
	      // process (cerr.Errors)
	      for _, err := range cerr.Errors {
	        el.handleError( errors.New("Check() " + err.Error() ) )
	      }
	      return false
	    } else {
	      el.handleError( errors.New("Check() bad cluster endpoints, which are not etcd servers. " + err.Error() ) )
	      return false
	    }
	  }
	  cancel()
	*/
	return true
}

func (el *Etcd) Put(value communsTypes.KeyValueType) error {
	var err error
	var jsonData []byte

	jsonData, err = json.Marshal(&value)
	if err != nil {
		el.handleError(err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), el.requestTimeOut)
	_, err = el.cli.Put(ctx, string(value.K), string(jsonData))
	if err != nil {
		el.handleError(err)
		return err
	}

	cancel()

	return err
}

func (el *Etcd) GetByPrefix(prefix []byte) (error, int, []communsTypes.KeyValueType) {
	var err error
	var resp *clientv3.GetResponse
	var value []communsTypes.KeyValueType

	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), el.requestTimeOut)
	resp, err = el.cli.Get(ctx, string(prefix), opts...)
	if err != nil {
		el.handleError(err)
		return err, 0, nil
	}

	defer cancel()

	value = make([]communsTypes.KeyValueType, resp.Count)

	if resp.Count == 0 {
		el.handleError(err)
		return nil, 0, value
	}

	for k := range resp.Kvs {
		value[k].K = resp.Kvs[k].Key

		err = json.Unmarshal(resp.Kvs[k].Value, &value[k])
		if err != nil {
			el.handleError(err)
			return err, 0, nil
		}
	}

	return nil, int(resp.Count), value
}

func (el *Etcd) Get(key []byte) (error, int, []communsTypes.KeyValueType) {
	var err error
	var resp *clientv3.GetResponse

	ctx, cancel := context.WithTimeout(context.Background(), el.requestTimeOut)
	resp, err = el.cli.Get(ctx, string(key))
	if err != nil {
		el.handleError(err)
		return err, 0, nil
	}

	defer cancel()

	if resp.Count == 0 {
		el.handleError(err)
		return nil, 0, nil
	}

	//fmt.Printf("count to make: %v\n", resp.Count)
	dataToGet := make([]communsTypes.KeyValueType, resp.Count)

	for k := range resp.Kvs {
		dataToGet[k].K = resp.Kvs[k].Key
		err = json.Unmarshal(resp.Kvs[k].Value, &dataToGet[k])
		if err != nil {
			el.handleError(err)
			return err, 0, nil
		}
	}

	return nil, int(resp.Count), dataToGet
}

func (el *Etcd) SetOnWatch(watchFunc func([]communsTypes.KeyValueType, []communsTypes.KeyValueType)) {
	el.onWatchFunc = watchFunc
}

func (el *Etcd) Watch(key []byte) {
	go el.watch(string(key))
}

func (el *Etcd) watch(key string) {
	var rch clientv3.WatchChan
	var err error

	rch = el.cli.Watch(context.Background(), key, clientv3.WithPrefix())
	for watchResp := range rch {
		if el.onWatchFunc != nil {
			eventLength := len(watchResp.Events)
			var newValue = make([]communsTypes.KeyValueType, eventLength)
			var oldValue = make([]communsTypes.KeyValueType, eventLength)

			for eventKey := range watchResp.Events {
				newValue[eventKey].T = []byte(watchResp.Events[eventKey].Type.String())
				newValue[eventKey].K = watchResp.Events[eventKey].Kv.Key
				err = json.Unmarshal(watchResp.Events[eventKey].Kv.Value, &newValue[eventKey])
				if err != nil {
					el.handleError(err)
					return
				}

				//oldValue[eventKey].T = []byte(watchResp.Events[eventKey].Type.String())
				//oldValue[eventKey].K = watchResp.Events[eventKey].PrevKv.Key
				//err = json.Unmarshal(watchResp.Events[eventKey].PrevKv.Value, &oldValue[eventKey])
				//if err != nil {
				//	el.handleError(err)
				//	return
				//}
			}

			el.onWatchFunc(newValue, oldValue)
		}
	}
}

func (el *Etcd) Delete(key []byte) error {
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), el.requestTimeOut)
	_, err = el.cli.Delete(ctx, string(key))
	cancel()

	return err
}

var PluginData Etcd
