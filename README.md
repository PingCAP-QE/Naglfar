# Naglfar

An automatic test pipeline for distributed systems.

## Install

### Prepare

- Tools
    - git
    - docker
    - kubectl
- Source Code
    ```bash
    git clone https://github.com/PingCAP-QE/Naglfar.git
    cd Naglfar
    ```

### Deploy

#### Single Node K8s

- Minikube

```bash
eval "$(minikube docker-env)"
make deploy
```

#### Multiple Node K8s

```bash
make docker-build
make docker-push
make deploy
```

Then execute `kubectl get crd | grep "naglfar.pingcap.com"` to inspect deployment. You should see:

```
machines.naglfar.pingcap.com
procchaos.naglfar.pingcap.com                
relationships.naglfar.pingcap.com           
testclustertopologies.naglfar.pingcap.com   
testresourcerequests.naglfar.pingcap.com    
testresources.naglfar.pingcap.com           
testworkflows.naglfar.pingcap.com           
testworkloads.naglfar.pingcap.com           
```

## Usage

### Add Host Machine

A host machine is the machine to allocate resources and execute job (cluster and workload).
The "resources" include memery, cpu cores and exclusive disks.

There are examples crds: [pipeline-examples](https://github.com/PingCAP-QE/Naglfar/tree/master/pipeline-examples).
You can see `machine.yaml`:

```yaml
apiVersion: naglfar.pingcap.com/v1
kind: Machine
metadata:
  name: 172.16.0.1
  labels:
    type: physical
spec:
  host: 172.16.0.1
  dockerPort: 2376
  dockerTLS: true
  reserve:
    memory: 2GB
    cores: 2
  exclusiveDisks:
    - /dev/nvme0n1
    - /dev/nvme1n1
```

You can apply a new machine by `kubectl apply -f <file path>`, then execute `kubectl get machine` to checkout applyment result. 

### Request for Resources

There are two steps to request resources: create your k8s `Namespace` and add a `TestResouceRequest`.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: test-tidb-cluster
---

apiVersion: naglfar.pingcap.com/v1
kind: TestResourceRequest
metadata:
  name: test-tidb-cluster
  namespace: test-tidb-cluster
spec:
  items:
    - name: n1
      spec:
        memory: 1GB
        cores: 2
        disks:
          disk1:
            kind: nvme
            size: 500GB
            mountPath: /disks1
    - name: n2
      spec:
        memory: 10GB
        cores: 4
        disks:
          disk1:
            kind: nvme
            size: 1TB
            mountPath: /disks1
    - name: n3
      spec:
        memory: 5GB
        cores: 2
```

The disks maps custom names to exclusive disks, it's unnecessary if you do not need exclusive disks. All resources are allocated automatically, naglfar system will find machines for them.

Now apply this file and checkout result by `kubectl get trr test-tidb-cluster -n test-tidb-cluster`.

### Define Cluster Topology
Once resource request is ready, you can define your cluster topology.

```yaml
apiVersion: naglfar.pingcap.com/v1
kind: TestClusterTopology
metadata:
  name: tidb-cluster
  namespace: test-tidb-cluster
spec:
  resourceRequest: test-tidb-cluster
  tidbCluster:
    global:
      deployDir: "/tidb-deploy"
      dataDir: "/tidb-data"
    serverConfigs:
      tidb: |-
        log.slow-threshold: 300
        binlog.enable: false
        binlog.ignore-error: false
      tikv: |-
        readpool.storage.use-unified-pool: false
        readpool.coprocessor.use-unified-pool: true
      pd: |-
        schedule.leader-schedule-limit: 4
        schedule.region-schedule-limit: 2048
        schedule.replica-schedule-limit: 64
    version:
      version: nightly
    control: n1
    tikv:
      - host: n1
        port: 20160
        statusPort: 20180
        deployDir: /disk1/deploy/tikv-20160
        dataDir: /disk1/data/tikv-20160
        logDir: /disk1/deploy/tikv-20160/log
    tidb:
      - host: n1
        deployDir: /disk1/deploy/tidb-4000
        logDir: /disk1/deploy/tidb-4000/log
    pd:
      - host: n1
        deployDir: /disk1/deploy/pd-2379
        dataDir: /disk1/data/pd-2379
        logDir: /disk1/deploy/pd-2379/log
    monitor:
      - host: n1
        deployDir: /disk1/deploy/prometheus-8249
        dataDir: /disk1/deploy/prometheus-8249/data
    grafana:
      - host: n2
        deployDir: /disk1/deploy/grafana-3000
    cdc:
      - host: n1
        port: 8300
        deployDir: /disk1/deploy/cdc
        logDir: /disk1/deploy/cdc
    tiflash:
      - host: n1
        tcpPort: 9000
        httpPort: 8123
        servicePort: 3930
        proxyPort: 20170
        proxyStatusPort: 20292
        metricsPort: 8234
        deployDir: /tidb-deploy/tiflash-9000
        dataDir: /tidb-data/tiflash-9000
        config: |-
          logger.level: info
        learnerConfig: |-
          log-level: info
      - host: n2
```

Apply this topology file, naglfar will deploy the cluster automatically. You can check the state by `kubectl get tct -n test-tidb-cluster`, it should print:

```log
NAMESPACE                           NAME           STATE     AGE
test-tidb-cluster                   tidb-cluster   pending   0h
```

### Create Workloads

Once the cluster is ready, you can create a series of workloads to start the tests. Following is a simple workload:

```yaml
apiVersion: naglfar.pingcap.com/v1
kind: TestWorkload
metadata:
  name: tidb-cluster-workload
  namespace: test-tidb-cluster
spec:
  clusterTopologies:
    - name: tidb-cluster
      aliasName: standard
  workloads:
    - name: sleep
      dockerContainer:
        resourceRequest:
          name: tidb-cluster
          node: n3
        image: "alpine"
        command:
          - sh
          - -c
          - sleep 100000
```

Apply this file then your workload will start! Although, this workload does nothing but sleep. In most cases your workload needs some information of cluster, for example, the address of tidb endpoint.

We provide SDK([github.com/PingCAP-QE/Naglfar/client](github.com/PingCAP-QE/Naglfar/client)) to access cluster information, but it only works on golang. If you want to use it on other languages, the according k8s sdk is more suitable (on bash, we have the kubectl!). 

With kubectl, you can describe the `n1` node of test resource by `kubectl describe tr n1 -n test-tidb-cluster`:

```yaml
Name:         n1
Namespace:    test-tidb-cluster
Labels:       <none>
Annotations:  <none>
API Version:  naglfar.pingcap.com/v1
Kind:         TestResource
Metadata:
  ...
Spec:
  ...
Status:
  ...
  Cluster IP:  10.0.2.213
  Exit Code:   0
  Exposed Ports:
    4000/tcp
    10080/tcp
    2379/tcp
    20180/tcp
    22/tcp
  Host IP:        172.16.0.1
  Image:          docker.io/mahjonp/base-image:latest
  Password:
  Phase:          running
  Port Bindings:  22/tcp:33115,2379/tcp:33114,4000/tcp:33113,10080/tcp:33112,20180/tcp:33111
  Ssh Port:       33115
  State:          ready
Events:           <none>
```

Then you can access the tidb endpoint by `10.0.2.213:4000`(docker swarm network) or `172.16.0.1:33113`(host machine network). You can even use ssh to login the root user by (`docker/insecure_key`)[https://github.com/PingCAP-QE/Naglfar/blob/master/docker/insecure_key].

### Apply ProcChaos

The `ProcChaos` means process chaos, you can define some patterns and rules to kill processes once or in a period.

```yaml
apiVersion: naglfar.pingcap.com/v1
kind: ProcChaos
metadata:
  name: tidb-cluster-chaos
  namespace: test-tidb-cluster
spec:
  request: test-tidb-cluster
  tasks:
    - pattern: tidb-server
      nodes: [n1]
      period: 3m
      killAll: true   
    - pattern: tikv-server
      nodes: [n1]
      period: 5m
    - pattern: pd-server
      nodes: [n1]
```

Applied this file, naglfar system will kill one process matching pattern 'pd-server' on node `n1`, then it will kill all processes matching pattern 'tidb-server' on node `n1` in period of 3 min and kill anyone matching pattern 'tikv-server' on node `n1` in period of 5 min. 


### Clean up
The `Workload` and `ProcChaos` can be delete straightly, just execute `kubectl delete workload tidb-cluster-workload -n test-tidb-cluster` or `kubectl delete procchaos tidb-cluster-chaos -n test-tidb-cluster`. Otherwise, you should delete the namespace to cleanup the whole test: `kubectl delete ns test-tidb-cluster`.

## K8S Plugin

### Install

```sh
curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/PingCAP-QE/Naglfar/master/scripts/kubectl-naglfar-installer.sh | sh
```

### Usages

* To fetch the workload logs, use the naglfar logs commands, as follows:

```sh
naglfar logs -n $ns --follow $name
```
