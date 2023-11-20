<!--
SPDX-License-Identifier: Apache-2.0
-->

# East-LB

East-LB is used for load balancing the data of user equipment and give it to the responsible UPFs. East-LB load balances the responce of the UE traffic which is decapsulated by UPFs and sent toward the internet. this components are built with two Kubernetes resources which are Pod and Service. Like other components its Pod contains a container which is running an app written in Golang. this apps run an HTTP Server, which is responsible for handling requests from UPFs. UPFs may send two kinds of requests and East-LB:

1.	when a UPF is deployed, register itself to West-LB and East-LB through HTTP POST request sending some info which are used to correctly route packets of user equipment toward UPFs. So, in this way West-LB and East-LB know how many active UPFs there are.

2.	when a UPF receives a Session Establishment Request, send the IP address of associated user equipment to and East-LB. in this way East-LB know each UE is assigned to which UPF, so it adds an iptables rule for that specific UE to mark the packets of that user equipment. This iptables rule matches UE’s IP and action of these iptables rules is just adding a mark to the packets that are used to correctly routing the packet using Policy-Based Routing.

### Config of East-LB for each interface

```
$ echo 101 lb-UPF101 >> /etc/iproute2/rt_tables
$ IP rule add fwmark 101 table lb-UPF101
$ IP route add 192.168.250.3 dev UPF101 table lb-UPF101
$ iptables -t mangle -A PREROUTING -d 172.250.0.1 -j MARK --set-mark 101
```

* lb-UPF101: rt_table name
* UPF101: interface of East-LB connected to UPF101
* 172.250.0.1 is IP of the UE
