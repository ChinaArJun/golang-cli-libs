// Brazilian Portuguese:
// Este é um plugin feito para funcionar como ferramenta de service discover em uma rede de microsserviços e se baseia
// no trabalho original de Ben Burkert - https://github.com/benburkert/dns, porém, o código original não foi previsto
// para que houvesse alterações na lista de serviços do DNS e eu terminei adaptando o trabalho dele para as minhas
// necessidades, transformando alguns tipos em type safe for thread e adicionando algumas funções a mais.
// Caso você necessite de um servidor de DNS feito com a necessidade de permitir a adição e remoção de registros com
// segurança, a minha versão está em https://github.com/helmutkemper/dns
//
// English:
// This is a plugin made to function as a tool of service discover in a network of microservices and is based on the
// original work of Ben Burkert - https://github.com/benburkert/dns however, the original code was not make to accept
// changes in the list of services of DNS and I adapt his work for my needs, transforming some types in a type safe for
// thread and adding some functions.
// If you need a server DNS made to allow additions and removal of records with security, my version is in
// https://github.com/helmutkemper/dns
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/helmutkemper/dns"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Dns struct {
	addressAndPort string
	serialNumber   int
	server         *dns.Server
}

type PluginDnsInterface interface {
	OnLoad(...interface{}) error
	Set(serviceList map[string]map[dns.Type][]dns.Record)
	GetAddressAndPort() string
	SetServiceByName(serviceName string, v map[dns.Type][]dns.Record)
	SetServiceBySRV(serviceName string, JSon []byte)
	AppendNewRegisterInServiceByName(serviceName string, v dns.Record)
	RemoveRegisterFromServiceByName(serviceName string, v dns.Record)
	RemoveServiceByName(serviceName string)
	Connect() error
	Test() error
}

func (el *Dns) handleError(err error) {
	if err != nil {
		_, fn, line, _ := runtime.Caller(1)
		log.Printf("[Ben Burkert DNS plugin error] %s:%d %v", fn, line, err)
	}
}

// on plugin load function
// conf[0] - string containing a json file path of configuration file
//
//   json example:
//   {
//     "addressAndPort": ":53",
//     "serialNumber": 1234
//   }
func (el *Dns) OnLoad(conf ...interface{}) error {
	var err error
	var fileContent []byte
	var jsonData map[string]interface{}
	var filePath = conf[0].([]interface{})[0].(string)
	var serialNumber int64

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

	el.addressAndPort = jsonData["addressAndPort"].(string)
	if el.addressAndPort == "" {
		err = errors.New("json addressAndPort key not found")
		el.handleError(err)
		return err
	}

	switch converted := jsonData["serialNumber"].(type) {
	case int:
		el.serialNumber = converted
	case string:
		serialNumber, err = strconv.ParseInt(converted, 10, 64)
		if err != nil {
			el.handleError(err)
			return err
		}
		el.serialNumber = int(serialNumber)
	}

	return nil
}

// set entire DNS records list
func (el *Dns) Set(serviceList map[string]map[dns.Type][]dns.Record) {
	el.server.Set(serviceList)
}

func (el *Dns) GetAddressAndPort() string {
	return el.addressAndPort
}

// set records for service name
func (el *Dns) SetServiceByName(serviceName string, v map[dns.Type][]dns.Record) {
	el.server.SetKey(serviceName, v)
}

func (el *Dns) SetServiceBySRV(serviceName string, JSon []byte) {
	var records []dns.SRV
	err := json.Unmarshal(JSon, &records)
	if err != nil {
		log.Printf("ganbiarra dns json error: %v\n", err)
		return
	}

	toSet := map[dns.Type][]dns.Record{}
	toSet[dns.TypeSRV] = make([]dns.Record, len(records))

	for k, v := range records {
		var avoidsProblemsWithPointers dns.SRV

		avoidsProblemsWithPointers.Target = v.Target
		avoidsProblemsWithPointers.Priority = v.Priority
		avoidsProblemsWithPointers.Port = v.Port
		avoidsProblemsWithPointers.Weight = v.Weight

		toSet[dns.TypeSRV][k] = &avoidsProblemsWithPointers
	}

	el.server.SetKey(serviceName, toSet)
}

// set or append records for service name
func (el *Dns) AppendNewRegisterInServiceByName(serviceName string, v dns.Record) {
	el.server.AppendRecordInKey(serviceName, v)
}

// remove register from service by service name
func (el *Dns) RemoveRegisterFromServiceByName(serviceName string, v dns.Record) {
	el.server.DeleteRecordInKey(serviceName, v)
}

// remove service key from records list
func (el *Dns) RemoveServiceByName(serviceName string) {
	el.server.DeleteKey(serviceName)
}

// start DNS service
func (el *Dns) Connect() error {
	var err error

	if el.addressAndPort == "" {
		err = errors.New("unable to connect. address and port configuration is missing.")
		el.handleError(err)
		return err
	}

	//fixme: ssl
	el.server = &dns.Server{
		Addr: el.addressAndPort,
		Handler: &dns.Zone{
			Origin: "tld.",
			TTL:    time.Hour,
			SOA: &dns.SOA{
				NS:     "dns.tld.",
				MBox:   "hostmaster.tld.",
				Serial: el.serialNumber,
			},
			RRs: dns.RRSet{},
		},
	}

	return el.server.ListenAndServe(context.Background())
}

func (el *Dns) Test() error {
	el.SetServiceByName("test.fake.service", map[dns.Type][]dns.Record{dns.TypeSRV: {&dns.SRV{Weight: 10, Priority: 0, Port: 8080, Target: "i.an.alive."}}})

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", net.JoinHostPort("127.0.0.1", "53535"))
		},
	}

	cname, srv, err := resolver.LookupSRV(context.Background(), "", "", "test.fake.service.tld.")
	if err != nil {
		return errors.New("Ben Bukert DNS test fail. Could not get IPs: " + err.Error() + "\n")
	}
	for _, s := range srv {
		fmt.Printf("%v: srv IN A Target: %s, Port: %v, Priority: %v, Weight: %v\n", cname, s.Target, s.Port, s.Priority, s.Weight)
	}

	el.RemoveServiceByName("test.fake.service")

	return nil
}

var PluginData Dns
