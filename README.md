# OpenStack Staus Exporter

This exporter used Gophercloud to query information
about the state of various OpenStack resources.

## Output 

```
# HELP openstack_lb_status Status of OpenStack Load Balancers
# TYPE openstack_lb_status gauge
openstack_lb_status{status="ACTIVE"} 370
openstack_lb_status{status="ERROR"} 13
# HELP openstack_router_status Status of OpenStack Routers
# TYPE openstack_router_status gauge
openstack_router_status{status="ACTIVE"} 734
openstack_router_status{status="ERROR"} 193 
# HELP openstack_server_status Status of OpenStack Servers
# TYPE openstack_server_status gauge
openstack_server_status{status="ACTIVE"} 5752
openstack_server_status{status="BUILD"} 6
openstack_server_status{status="ERROR"} 29
openstack_server_status{status="REBOOT"} 1
openstack_server_status{status="SHUTOFF"} 300
openstack_server_status{status="SUSPENDED"} 5
openstack_server_status{status="VERIFY_RESIZE"} 20
# HELP openstack_up OpenStack Status Collection Operational
# TYPE openstack_up gauge
openstack_up 1
# HELP openstack_volume_status Status of OpenStack Volumes
# TYPE openstack_volume_status gauge
openstack_volume_status{status="available"} 84
openstack_volume_status{status="detaching"} 1
```
