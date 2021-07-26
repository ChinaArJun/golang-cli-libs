#gRPC

> This is a live repository used for testing, if any folder is not documented here, don't use its content, it
is not ready and can be discarded without notice.

Nowadays, there has been a lot of talk about microservices, however, the vast majority of people talk a lot and say nothing.
This repository is a means of making all my studies on microservices and gRPC public for discussion with the
developer community

> As a prerequisite, you will need docker and compose installed.
> I do my tests in docker compose version 1.21.2.

To find out what your version is, run the command below.

```console

  $ docker-compose -v
  > docker-compose version 1.21.2, build a133471
  
```

## Hello Word

> Folder: ./1_helloWord

The website [gRPC.io](https://grpc.io/) has a series of simple examples on gRPC and the simplest example of all is
**Hello Word**, an example client/server communicating and sending a welcome message.

In my case, I would like to know how pure **gRPC** would behave in case of failure and that's why I created the project
**[pygocentrus](https://github.com/helmutkemper/pygocentrus)**, a school of piranhas to be inserted into the network flow
with the intention of causing purposeful errors in the data and simulating the failures of the real world production environment.

Therefore, the client/server communication was as described in the image below

![pygocentrus attack](https://github.com/helmutkemper/pygocentrus/blob/master/img/grpc_and_pygocentrus.png)

To run the first example, enter the **./1_helloWord** folder and run the command below

```console

  /1_helloWord$ docker-compose up
  
```

In some cases, client/server communication will be normal and in some cases there will be a pygocentrus attack, the
feared red piranhas common in South America.

If communication works normally, docker-compose will print output like this:

```console

  Creating network "1_helloword_grpc_net" with the default driver
  Creating 1_helloword_pygocentrus_1 ... done
  Creating 1_helloword_golang-grpc-server_1 ... done
  Creating 1_helloword_golang-grpc-client_1 ... done
  Attaching to 1_helloword_pygocentrus_1, 1_helloword_golang-grpc-server_1, 1_helloword_golang-grpc-client_1
  pygocentrus_1 | pygocentrus listen(:50051)
  pygocentrus_1 | pygocentrus dial(:172.28.0.5:50051)
  golang-grpc-server_1 | gRPC server listen(:50051)
  golang-grpc-client_1 | gRPC client dial(172.28.0.2:50051)
  golang-grpc-server_1 | 2019/03/09 13:14:05 Received: gRPC Dev
  golang-grpc-client_1 | 2019/03/09 13:14:05 Greeting: Hello gRPC Dev
  1_helloword_golang-grpc-client_1 exited with code 0

```

As you can see, there was normal client/server communication and everything worked as expected. You can see this in the
message exchange, as in the lines below:

```console

  golang-grpc-server_1 | 2019/03/09 13:14:05 Received: gRPC Dev
  golang-grpc-client_1 | 2019/03/09 13:14:05 Greeting: Hello gRPC Dev

```

If the **[pygocentrus](https://github.com/helmutkemper/pygocentrus)** project decides to give its random bites, the
docker-compose will print the output below:

```console

  Starting 1_helloword_pygocentrus_1 ... done
  Starting 1_helloword_golang-grpc-server_1 ... done
  Starting 1_helloword_golang-grpc-client_1 ... done
  Attaching to 1_helloword_pygocentrus_1, 1_helloword_golang-grpc-server_1, 1_helloword_golang-grpc-client_1
  pygocentrus_1 | pygocentrus listen(:50051)
  pygocentrus_1 | pygocentrus dial(:172.28.0.5:50051)
  golang-grpc-server_1 | gRPC server listen(:50051)
  pygocentrus_1 | 2019/03/09 13:14:14 pygocentrus attack
  golang-grpc-client_1 | gRPC client dial(172.28.0.2:50051)
  golang-grpc-client_1 | 2019/03/09 13:14:14 could not greet: rpc error: code = Unavailable desc = transport is closing
  1_helloword_golang-grpc-client_1 exited with code 0
    
```

It's been set up to swap some bytes randomly and mess up the communication, so you'll see the line
below:

```console

  golang-grpc-client_1 | 2019/03/09 13:14:14 could not greet: rpc error: code = Unavailable desc = transport is closing
    
```

If you're new to docker-compose, to quit, press **ctrl** + **C**.

## Service Discover

The basis of the microservices architecture is **Service Discover**, a network entity in charge of knowing all the
services and inform the other services how their respective servers can be accessed.

At this point, there is a lot of freedom to choose a **Service Discover** and there is very little documentation on
the subject, then, I noticed three popular patterns:

  *ETCD
  * gossip
  * mDNS

The **[ETCD](https://coreos.com/etcd/)**, an open source key/value database, where they are usually raised
three replicas on different servers and these replicas communicate with each other and synchronize the data in a way
automatic.
 
**[mDNS](https://en.wikipedia.org/wiki/Multicast_DNS)** is the most elegant way to do a **Service Discover**,
in my opinion, and it's very easy for many programming languages ​​to ask the **mDNS** of the network where the services
they are.

Simply put, **mDNS** and **DNS** are the same thing, except **mDNS** is made to run on a local network and the
**DNS** is global on the internet.

By default, DNS works on port **53** and if you use any programming language inside a container, it
will look for the external **DNS**, so be careful.

Below are two examples of how to query a **Service Discover** for **DNS**/**mDNS** on the local network.

node.js code:
```javascript

const { Resolve } = require('dns');

const resolve = new Resolver();
resolver.setServers(['127.0.0.1:53']);
resolver.resolveSrv('service.tld.', function (err, addresses, family) {
    console.log(addresses);
});

```

Response:
```json

[
  { "name": "192.168.0.1", "port": 8080, "priority": 0, "weight": 10 },
  { "name": "192.168.0.2", "port": 8081, "priority": 0, "weight": 50 }
]

```

Golang Code:
```golang

package main

import (
  "context"
  "fmt"
  "net"
  "you"
)

func main() {
  var resolve *net.Resolver
  resolve = &net.Resolve{
    PreferGo: true,
    Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
      d := net.Dialer{}
      return d.DialContext(ctx, "udp", net.JoinHostPort("127.0.0.1", "53"))
    },
  }
  
  cname, srv, err := resolve.LookupSRV(context.Background(),"", "", "service.discover.tld.")
  if err != nil {
    fmt.Fprintf(os.Stderr, "Could not get IPs: %v\n", err)
    os.Exit(1)
  }
  for _, s := range srv {
    fmt.Printf("srv IN A Target: %s, Port: %v, Priority: %v, Weight: %v\n", s.Target, s.Port, s.Priority, s.Weight)
  }
}

```

Response:
```console

  srv IN A Target: 192.168.0.1., Port: 8080, Priority: 0, Weight: 10
  srv IN A Target: 192.168.0.2., Port: 8081, Priority: 0, Weight: 50

```

As you can see, in the answers there are parameters:

  name/target - It is a string containing the address of the service and usually ending in a dot, however, the dot can be
  automatically removed by the class you use. It's good to test before using;
  port - integer containing the service port;
  priority - unsigned integer containing the priority. Remember that the lower, the higher the priority;
  weight - unsigned int used as weight for services with the same priority.

In practice, you will receive the list of addresses, choose the service with the lowest priority and make a random choice.
using the **weight** parameter as the weight.




In my case, I

## go-micro

**go-micro** is a framework written in go and is intended to facilitate the use of micro services and make it much easier
use of tools such as service discover so that services communicate automatically, without configuration
additional.

This was a first test, where several services came up without the need for further docker-compose settings,
thing that did not happen with pure **gRPC**.

```
  docker-compose -f go-micro.yml up --scale go-micro-server=10 --scale go-micro-client=50 --build
```

## etcd

Just upload etcd in compose and it's there for testing.

```
  docker-compose -f etcd.yml up --build
```

## mobyServer

The **mobby** project is a project designed to work with containers in golang and allows full control of
containers by golang applications.

This was a test of how to use the code.

Basically, it looks for all containers running on the machine, stops and then puts the ghost blog to run on the
port 8080.

```
  
```

## Request cancellation

This is an example of how to cancel an ongoing request.

```
  docker-compose -f cancellation.yml up --scale golang-grpc-client=10 --build
```

**Cancel execution**
```
  Ctrl+C
```

**Take off the docker and active containers**
```
  docker-compose -f helloWord.yml down
```