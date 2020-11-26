## Naglfar

An automatic test pipeline for distributed systems.

## How to install plugin

```sh
curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/PingCAP-QE/Naglfar/master/scripts/kubectl-naglfar-installer.sh | sh
```

## Usages

* To fetch the workload logs, use the naglfar logs commands, as follows:

```sh
naglfar logs -n $ns --follow $name
```
