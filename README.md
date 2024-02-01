背景.

国内有服务器要装k8s和docker, 以及要装python的库, 很可惜都被半墙不墙的状态搞得有时候可用有时候不可用.

全局代理, 浪费流量, 且有些业务不能走代理.

iptables分流, 或者ipset分流中国和国外ip, 感觉略微麻烦.

遂根据k8s和docker定制了一个方案.

---

SNIProxy用的是这个: **https://github.com/XIU2/SNIProxy**

---

首先得有个socks5的代理, 可以是ssh, 例如

```
ssh [你国外的服务器] -D 1080
```

---

然后开启dns服务器

```
./dnsserver.linux.amd64 -file blacklist
```

其中blacklist是一行一个的域名, 命中域名的就解析到指定的ip(sni-proxy的ip), 否则就返回上游的结果

---

再开启sni-proxy

```
./sni-proxy.linux.amd64 -c config.yaml
```

配置文件不用改.

---

再把本机的dns服务器指向本机

```
echo nameserver 127.0.0.1 > /etc/resolv.conf
```

---

即可
