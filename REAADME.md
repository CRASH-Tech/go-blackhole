### config.yaml
```
bgp:
  router_id: 10.171.121.15
  local_as: 65534
  neighbors:
    - peer_address: 10.171.121.254
      peer_as: 65533

feeds:
 - url: http://10.0.0.1/iplist_exports/ip2ban_1.txt
   community: 65534:666
   refresh_interval: 5s
 - url: http://10.0.0.1/iplist_exports/ip2ban_2.txt
   community: 65534:666
   refresh_interval: 5s
 - url: http://10.0.0.1/iplist_exports/ip2ban_3.txt
   community: 65534:666
   refresh_interval: 5s

web:
 listen: "0.0.0.0:8080"

```