package main

import (
	"fmt"
	"github.com/libvirt/libvirt-go"
)

func domainEventCallback(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventLifecycle) {
	name, _ := d.GetName()
	eventType := event.Event

	switch eventType {
	case libvirt.DOMAIN_EVENT_DEFINED:
		fmt.Printf("Domain '%s' defined\n", name)
	case libvirt.DOMAIN_EVENT_STARTED:
		fmt.Printf("Domain '%s' started\n", name)
	case libvirt.DOMAIN_EVENT_SUSPENDED:
		fmt.Printf("Domain '%s' suspended\n", name)
	case libvirt.DOMAIN_EVENT_RESUMED:
		fmt.Printf("Domain '%s' resumed\n", name)
	case libvirt.DOMAIN_EVENT_STOPPED:
		fmt.Printf("Domain '%s' stopped\n", name)
	case libvirt.DOMAIN_EVENT_SHUTDOWN:
		fmt.Printf("Domain '%s' is being shut down\n", name)
	case libvirt.DOMAIN_EVENT_PMSUSPENDED:
		fmt.Printf("Domain '%s' PMSuspended\n", name)
	case libvirt.DOMAIN_EVENT_CRASHED:
		fmt.Printf("Domain '%s' crashed\n", name)
	case libvirt.DOMAIN_EVENT_UNDEFINED:
		fmt.Printf("Domain '%s' undefined\n", name)
	default:
		fmt.Printf("Domain '%s': unhandled event %d\n", name, eventType)
	}
}

func domainEventBlockJobCallback(c *libvirt.Connect, d *libvirt.Domain, event *libvirt.DomainEventBlockJob) {
	name, _ := d.GetName()
	fmt.Printf("Domain '%s' block job event for disk : %+v\n", name, event)
}

// 在此处添加更多回调函数以处理其他事件...

func main() {
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		fmt.Println("Failed to connect to hypervisor:", err)
		return
	}
	defer conn.Close()

	if err := libvirt.EventRegisterDefaultImpl(); err != nil {
		fmt.Println("Failed to register event implementation:", err)
		return
	}

	domains, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		fmt.Println("Failed to list all active domains:", err)
		return
	}

	for _, domain := range domains {
		// 注册生命周期事件回调
		lifecycleCallbackID, err := conn.DomainEventLifecycleRegister(&domain, domainEventCallback)
		if err != nil {
			name, _ := domain.GetName()
			fmt.Printf("Failed to register lifecycle events for domain '%s': %v\n", name, err)
		}
		defer conn.DomainEventDeregister(lifecycleCallbackID)

		// 注册块作业事件回调
		blockJobCallbackID, err := conn.DomainEventBlockJobRegister(&domain, domainEventBlockJobCallback)
		if err != nil {
			name, _ := domain.GetName()
			fmt.Printf("Failed to register block job events for domain '%s': %v\n", name, err)
		}
		defer conn.DomainEventDeregister(blockJobCallbackID)

		// 在此处注册其他事件...
		domain.Free()
	}

	for {
		if err := libvirt.EventRunDefaultImpl(); err != nil {
			fmt.Println("Error running event loop:", err)
			break
		}
	}
}
