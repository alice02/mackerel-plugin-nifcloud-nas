mackerel-plugin-nifcloud-nas
================================

NIFCLOUD NAS custom metrics plugin for mackerel.io agent.

## Install

### Binary

You can download binary from [release](https://github.com/aokumasan/mackerel-plugin-nifcloud-nas/releases) page.

### Docker image

You can use `mackerel-plugin-nifcloud-nas` built-in mackerel-agent [image](https://hub.docker.com/r/aokumasan/mackerel-agent).

```sh
docker pull aokumasan/mackerel-agent
```

## Usage

```
mackerel-plugin-nifcloud-nas -identifier=<nas-instance-identifier> -access-key-id=<id> secret-access-key=<key> -region=<jp-east-1 or jp-east-2 or jp-east-3 or jp-east-4 or jp-west-1> [-tempfile=<tempfile>] [-metric-key-prefix=<prefix>] [-metric-label-prefix=<label-prefix>]"
```


## Example of mackerel-agent.conf

```
[plugin.metrics.nas]
command = "mackerel-plugin-nifcloud-nas -identifier=sample -access-key-id=your_access_key -secret-access-key=your_secret_key -region=jp-east-1"
```
