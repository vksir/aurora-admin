# dst_run

## Install

```shell
curl -OL https://raw.githubusercontent.com/vksir/dst-run/master/scripts/setup.sh && bash setup.sh install
```

## Run


```shell
systemctl start dstrun
```

```docker
docker run -d \
    -v /opt/aurora-admin/aurora-admin:/root/aurora-admin \
    -v /opt/aurora-admin/dontstarve_clusters:/root/.klei/DoNotStarveTogether \
    -p 5800:5800 \
    -p 10999:10999 \
    --restart=always \
    ubuntu:latest
   

docker run -d \
    -v /opt/aurora-admin/aurora-admin/:/root/aurora-admin \
    -v /opt/aurora-admin/dontstarve_clusters/:/root/.klei/DoNotStarveTogether \
    -v /opt/docker-helper/:/opt/docker-helper \
    -p 5800:5800 \
    -p 10999:10999/udp \
    -p 10998:10998/udp \
    --restart=always \
    ubuntu:latest \
    /opt/docker-helper/loop.sh

docker ps -a | grep ubuntu:latest | awk '{print $1}' | xargs docker rm -f

apt update
apt-get install curl lib32gcc-s1 libcurl3-gnutls -y
mkdir -p ~/aurora-admin/service/steamcmd
cd ~/aurora-admin/service/steamcmd
curl -OL https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz
tar -xvzf steamcmd_linux.tar.gz
rm steamcmd_linux.tar.gz

```

## Dev

### Swag

```shell
swag init -g ./internal/app/http.go -o ./docs/ -p snakecase
```

### Ent

```shell
go generate ./ent
```

## Support

Thanks to [JetBrains](https://www.jetbrains.com/?from=neutron-star) for providing free licenses for this open source project.

[![JetBrains](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://www.jetbrains.com/?from=neutron-star)