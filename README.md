## Naglfar

An automatic test pipeline for distributed systems.

### Install

#### Prepare

- Tools
    - git
    - docker
    - kubectl
    - kustomize
    - [controller-gen](https://github.com/kubernetes-sigs/controller-tools/tree/master/cmd/controller-gen)
- Source Code
    ```bash
    git clone https://github.com/PingCAP-QE/Naglfar.git
    cd Naglfar
    ```

#### Deploy

##### Single Node K8s

- Minikube

```bash
eval "$(minikube docker-env)"
make deploy
```

##### Multiple Node K8s

```bash
make docker-build
make docker-push
make deploy
```

Then execute `kubectl get crd | grep "naglfar.pingcap.com"` to inspect deployment. You should see:

```
machines.naglfar.pingcap.com                
relationships.naglfar.pingcap.com           
testclustertopologies.naglfar.pingcap.com   
testresourcerequests.naglfar.pingcap.com    
testresources.naglfar.pingcap.com           
testworkflows.naglfar.pingcap.com           
testworkloads.naglfar.pingcap.com           
```

### Usage

#### Add Host Machine

A host machine is the machine to allocate resources and execute job (cluster and workload).
The "resources" includes memery, cpu cores and exclusive disks.

There are examples crds: [pipeline-examples](https://github.com/PingCAP-QE/Naglfar/tree/master/pipeline-examples).