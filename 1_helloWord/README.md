# Hello Word gRPC width Capibaribe Reverse Proxy

This example uses the example contained in the [grpc.io](https://grpc.io/) site with the help of the reverse 
proxy capibaribe to intercept the data and induce random errors in communication.

```yaml


version: 1.0

capibaribe:

  affluentRiverNameProject:
    listen: 127.0.0.1:50051
    debugServerEnable: false
    ssl:
      enabled: false
    pygocentrus:
        enabled: true
        delay:
          rate: 0.0
          min: 2000000
          max: 5000000
        dontRespond:
          rate: 0.0
          min: 2000000
          max: 5000000
        changeLength: 0.0
        changeContent:
          changeRateMin: 0.0
          changeRateMax: 0.0
          changeBytesMin: 1
          changeBytesMax: 10
          rate: 1.0
        deleteContent: 0.0
    proxy:
      - ignorePort: true
        host: grpc.127.0.0.1
        loadBalancing: roundRobin
        path: /
        maxAttemptToRescueLoop: 10
        servers:
          - host: 127.0.0.1:50052
            weight: 1
            overLoad: 1000000

```