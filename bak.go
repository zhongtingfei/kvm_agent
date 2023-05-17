package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	libvirt "github.com/libvirt/libvirt-go"
	"log"
	"net/http"
)

type VirtualMachine struct {
	ID        string `json:"id"`
	CPU       int    `json:"cpu"`
	Memory    uint64 `json:"memory"`
	DiskCount int    `json:"diskCount"`
	DiskSize  uint64 `json:"diskSize"`
	UUID      string `json:"uuid"`
	IPAddress string `json:"ipAddress"`
	Status    string `json:"status"`
}

var (
	conn     *libvirt.Connect
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// 允许所有来源的WebSocket连接，您可以根据需求进行调整
			return true
		},
	}
)

func main() {
	// 连接到本地libvirt daemon
	var err error
	conn, err = libvirt.NewConnect("qemu:///system")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	router := mux.NewRouter()
	router.HandleFunc("/api/vms", getVMs).Methods("GET")
	router.HandleFunc("/ws", handleWebSocket)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func getVMs(w http.ResponseWriter, r *http.Request) {
	// 获取所有虚拟机
	doms, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_RUNNING)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	vms := make([]*VirtualMachine, 0, len(doms))
	for _, dom := range doms {
		vm := &VirtualMachine{
			ID:     dom.GetName(),
			Status: "running",
		}

		// 获取虚拟机的CPU和内存信息
		info, err := dom.GetInfo()
		if err != nil {
			log.Println(err)
			continue
		}
		vm.CPU = int(info.NrVirtCpu)
		vm.Memory = info.Memory

		// 获取虚拟机的磁盘信息
		disks, err := dom.ListAllStorageVolumes(0)
		if err != nil {
			log.Println(err)
			continue
		}
		vm.DiskCount = len(disks)

		var totalDiskSize uint64
		for _, disk := range disks {
			size, err := disk.GetInfo()
			if err != nil {
				log.Println(err)
				continue
			}
			totalDiskSize += size.Capacity
		}
		vm.DiskSize = totalDiskSize

		// 获取虚拟机的UUID
		uuid, err := dom.GetUUIDString()
		if err != nil {
			log.Println(err)
			continue
		}
		vm.UUID = uuid

		// 获取虚拟机的IP地址
		ipAddress, err := getIPAddress(dom)
		if err != nil {
			log.Println(err)
		}
		vm.IPAddress = ipAddress

		vms = append(vms, vm)
	}

	// 将虚拟机列表编码为JSON并发送给客户端
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vms)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	// 获取所有虚拟机
	doms, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_RUNNING)
	if err != nil {
		log.Println(err)
		return
	}

	// 循环发送虚拟机信息到WebSocket客户端
	for _, dom := range doms {
		vm := &VirtualMachine{
			ID:     dom.GetName(),
			Status: "running",
		}

		// 获取虚拟机的CPU和内存信息
		info, err := dom.GetInfo()
		if err != nil {
			log.Println(err)
			continue
		}
		vm.CPU = int(info.NrVirtCpu)
		vm.Memory = info.Memory

		// 获取虚拟机的磁盘信息
		disks, err := dom.ListAllStorageVolumes(0)
		if err != nil {
			log.Println(err)
			continue
		}
		vm.DiskCount = len(disks)

		var totalDiskSize uint64
		for _, disk := range disks {
			size, err := disk.GetInfo()
			if err != nil {
				log.Println(err)
				continue
			}
			totalDiskSize += size.Capacity
		}
		vm.DiskSize = totalDiskSize

		// 获取虚拟机的UUID
		uuid, err := dom.GetUUIDString()
		if err != nil {
			log.Println(err)
			continue
		}
		vm.UUID = uuid

		// 获取虚拟机的IP地址
		ipAddress, err := getIPAddress(dom)
		if err != nil {
			log.Println(err)
		}
		vm.IPAddress = ipAddress

		// 将虚拟机信息发送给WebSocket客户端
		err = conn.WriteJSON(vm)
		if err != nil {
			log.Println("WebSocket write failed:", err)
			return
		}
	}
}

func getIPAddress(dom *libvirt.Domain) (string, error) {
	ifaces, err := dom.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		for _, addr := range iface.Addrs {
			if addr.Type == libvirt.DOMAIN_ADDR_TYPE_IPV4 && addr.Addr != "" {
				return addr.Addr, nil
			}
		}
	}

	return "", nil
}
