<!--
SPDX-License-Identifier: Apache-2.0
-->

# Core-LB

Core-LB is used for load balancing the data of user equipment and give it to the responsible UPFs. Core-LB load balances the responce of the UE traffic which is decapsulated by UPFs and sent toward the internet. this components are built with two Kubernetes resources which are Pod and Service. Like other components its Pod contains a container which is running an app written in Golang. this apps run an HTTP Server, which is responsible for handling requests from UPFs. UPFs may send two kinds of requests and Core-LB:

1.	when a UPF is deployed, register itself to Access-LB and Core-LB through HTTP POST request sending some info which are used to correctly route packets of user equipment toward UPFs. So, in this way Access-LB and Core-LB know how many active UPFs there are.

2.	when a UPF receives a Session Establishment Request, send the IP address of associated user equipment to and Core-LB. in this way Core-LB know each UE is assigned to which UPF, so it adds an iptables rule for that specific UE to mark the packets of that user equipment. This iptables rule matches UE’s IP and action of these iptables rules is just adding a mark to the packets that are used to correctly routing the packet using Policy-Based Routing.

### Config of Core-LB for each interface

```
$ echo 101 lb-UPF101 >> /etc/iproute2/rt_tables
$ IP rule add fwmark 101 table lb-UPF101
$ IP route add 192.168.250.3 dev UPF101 table lb-UPF101
$ iptables -t mangle -A PREROUTING -d 192.168.252.3 -j MARK --set-mark 101
```

* lb-UPF101: rt_table name
* UPF101: interface of Core-LB connected to UPF101
