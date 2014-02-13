package platform

import (
	bosherr "bosh/errors"
	boshcmd "bosh/platform/commands"
	boshstats "bosh/platform/stats"
	boshvitals "bosh/platform/vitals"
	boshsettings "bosh/settings"
	boshdir "bosh/settings/directories"
	boshsys "bosh/system"
	"bytes"
	"path/filepath"
	"text/template"
	"time"
)

type centos struct {
	linux           linux
	arpWaitInterval time.Duration
}

func NewCentosPlatform(
	linux linux,
	arpWaitInterval time.Duration,
) (platform centos) {
	platform.linux = linux
	platform.arpWaitInterval = arpWaitInterval
	return
}

func (p centos) GetFs() (fs boshsys.FileSystem) {
	return p.linux.GetFs()
}

func (p centos) GetRunner() (runner boshsys.CmdRunner) {
	return p.linux.GetRunner()
}

func (p centos) GetStatsCollector() (statsCollector boshstats.StatsCollector) {
	return p.linux.GetStatsCollector()
}

func (p centos) GetCompressor() (runner boshcmd.Compressor) {
	return p.linux.GetCompressor()
}

func (p centos) GetCopier() (runner boshcmd.Copier) {
	return p.linux.GetCopier()
}

func (p centos) GetDirProvider() (dirProvider boshdir.DirectoriesProvider) {
	return p.linux.GetDirProvider()
}

func (p centos) GetVitalsService() (service boshvitals.Service) {
	return p.linux.GetVitalsService()
}

func (p centos) SetupRuntimeConfiguration() (err error) {
	return p.linux.SetupRuntimeConfiguration()
}

func (p centos) CreateUser(username, password, basePath string) (err error) {
	return p.linux.CreateUser(username, password, basePath)
}

func (p centos) AddUserToGroups(username string, groups []string) (err error) {
	return p.linux.AddUserToGroups(username, groups)
}

func (p centos) DeleteEphemeralUsersMatching(reg string) (err error) {
	return p.linux.DeleteEphemeralUsersMatching(reg)
}

func (p centos) SetupSsh(publicKey, username string) (err error) {
	return p.linux.SetupSsh(publicKey, username)
}

func (p centos) SetUserPassword(user, encryptedPwd string) (err error) {
	return p.linux.SetUserPassword(user, encryptedPwd)
}

func (p centos) SetupHostname(hostname string) (err error) {
	return p.linux.SetupHostname(hostname)
}

func (p centos) SetupLogrotate(groupName, basePath, size string) (err error) {
	return p.linux.SetupLogrotate(groupName, basePath, size)
}

func (p centos) SetTimeWithNtpServers(servers []string) (err error) {
	return p.linux.SetTimeWithNtpServers(servers)
}

func (p centos) SetupEphemeralDiskWithPath(realPath string) (err error) {
	return p.linux.SetupEphemeralDiskWithPath(realPath)
}

func (p centos) SetupTmpDir() (err error) {
	return p.linux.SetupTmpDir()
}

func (p centos) MountPersistentDisk(devicePath, mountPoint string) (err error) {
	return p.linux.MountPersistentDisk(devicePath, mountPoint)
}

func (p centos) UnmountPersistentDisk(devicePath string) (didUnmount bool, err error) {
	return p.linux.UnmountPersistentDisk(devicePath)
}

func (p centos) NormalizeDiskPath(devicePath string) (realPath string, found bool) {
	return p.linux.NormalizeDiskPath(devicePath)
}

func (p centos) GetFileContentsFromCDROM(fileName string) (contents []byte, err error) {
	return p.linux.GetFileContentsFromCDROM(fileName)
}

func (p centos) IsMountPoint(path string) (result bool, err error) {
	return p.linux.IsMountPoint(path)
}

func (p centos) MigratePersistentDisk(fromMountPoint, toMountPoint string) (err error) {
	return p.linux.MigratePersistentDisk(fromMountPoint, toMountPoint)
}

func (p centos) IsDevicePathMounted(path string) (result bool, err error) {
	return p.linux.IsDevicePathMounted(path)
}

func (p centos) StartMonit() (err error) {
	return p.linux.StartMonit()
}

func (p centos) SetupMonitUser() (err error) {
	return p.linux.SetupMonitUser()
}

func (p centos) GetMonitCredentials() (username, password string, err error) {
	return p.linux.GetMonitCredentials()
}

func (p centos) getDnsServers(networks boshsettings.Networks) (dnsServers []string) {
	dnsNetwork, found := networks.DefaultNetworkFor("dns")
	if found {
		for i := len(dnsNetwork.Dns) - 1; i >= 0; i-- {
			dnsServers = append(dnsServers, dnsNetwork.Dns[i])
		}
	}

	return
}

func (p centos) SetupDhcp(networks boshsettings.Networks) (err error) {
	dnsServers := []string{}
	dnsNetwork, found := networks.DefaultNetworkFor("dns")
	if found {
		for i := len(dnsNetwork.Dns) - 1; i >= 0; i-- {
			dnsServers = append(dnsServers, dnsNetwork.Dns[i])
		}
	}

	type dhcpConfigArg struct {
		DnsServers []string
	}

	buffer := bytes.NewBuffer([]byte{})
	t := template.Must(template.New("dhcp-config").Parse(CENTOS_DHCP_CONFIG_TEMPLATE))

	err = t.Execute(buffer, dhcpConfigArg{dnsServers})
	if err != nil {
		err = bosherr.WrapError(err, "Generating config from template")
		return
	}

	written, err := p.linux.GetFs().WriteToFile("/etc/dhcp/dhclient.conf", buffer.String())
	if err != nil {
		err = bosherr.WrapError(err, "Writing to /etc/dhcp/dhclient.conf")
		return
	}

	if written {
		// Ignore errors here, just run the commands
		p.linux.GetRunner().RunCommand("service", "network", "restart")
	}

	return
}

// DHCP Config file - /etc/dhcp3/dhclient.conf
const CENTOS_DHCP_CONFIG_TEMPLATE = `# Generated by bosh-agent

option rfc3442-classless-static-routes code 121 = array of unsigned integer 8;

send host-name "<hostname>";

request subnet-mask, broadcast-address, time-offset, routers,
	domain-name, domain-name-servers, domain-search, host-name,
	netbios-name-servers, netbios-scope, interface-mtu,
	rfc3442-classless-static-routes, ntp-servers;

{{ range .DnsServers }}prepend domain-name-servers {{ . }};
{{ end }}`

func (p centos) SetupManualNetworking(networks boshsettings.Networks) (err error) {
	modifiedNetworks, err := p.writeIfcfgs(networks)
	if err != nil {
		err = bosherr.WrapError(err, "Writing network interfaces")
		return
	}

	p.restartNetwork()

	err = p.writeResolvConf(networks)
	if err != nil {
		err = bosherr.WrapError(err, "Writing resolv.conf")
		return
	}

	go p.gratuitiousArp(modifiedNetworks)

	return
}

func (p centos) gratuitiousArp(networks []customNetwork) {
	for i := 0; i < 6; i++ {
		for _, network := range networks {
			for !p.linux.GetFs().FileExists(filepath.Join("/sys/class/net", network.Interface)) {
				time.Sleep(100 * time.Millisecond)
			}

			p.linux.GetRunner().RunCommand("arping", "-c", "1", "-U", "-I", network.Interface, network.Ip)
			time.Sleep(p.arpWaitInterval)
		}
	}
	return
}

func (p centos) writeIfcfgs(networks boshsettings.Networks) (modifiedNetworks []customNetwork, err error) {
	macAddresses, err := p.detectMacAddresses()
	if err != nil {
		err = bosherr.WrapError(err, "Detecting mac addresses")
		return
	}

	for _, aNet := range networks {
		var network, broadcast string
		network, broadcast, err = boshsys.CalculateNetworkAndBroadcast(aNet.Ip, aNet.Netmask)
		if err != nil {
			err = bosherr.WrapError(err, "Calculating network and broadcast")
			return
		}

		newNet := customNetwork{
			aNet,
			macAddresses[aNet.Mac],
			network,
			broadcast,
			true,
		}
		modifiedNetworks = append(modifiedNetworks, newNet)

		buffer := bytes.NewBuffer([]byte{})
		t := template.Must(template.New("ifcfg").Parse(CENTOS_IFCFG_TEMPLATE))

		err = t.Execute(buffer, newNet)
		if err != nil {
			err = bosherr.WrapError(err, "Generating config from template")
			return
		}

		_, err = p.linux.GetFs().WriteToFile(filepath.Join("/etc/sysconfig/network-scripts", "ifcfg-"+newNet.Interface), buffer.String())
		if err != nil {
			err = bosherr.WrapError(err, "Writing to /etc/sysconfig/network-scripts")
			return
		}
	}

	return
}

const CENTOS_IFCFG_TEMPLATE = `DEVICE={{ .Interface }}
BOOTPROTO=static
IPADDR={{ .Ip }}
NETMASK={{ .Netmask }}
BROADCAST={{ .Broadcast }}
{{ if .HasDefaultGateway }}GATEWAY={{ .Gateway }}{{ end }}
ONBOOT=yes`

func (p centos) writeResolvConf(networks boshsettings.Networks) (err error) {
	buffer := bytes.NewBuffer([]byte{})
	t := template.Must(template.New("resolv-conf").Parse(CENTOS_RESOLV_CONF_TEMPLATE))

	dnsServers := p.getDnsServers(networks)
	dnsServersArg := dnsConfigArg{dnsServers}
	err = t.Execute(buffer, dnsServersArg)
	if err != nil {
		err = bosherr.WrapError(err, "Generating config from template")
		return
	}

	_, err = p.linux.GetFs().WriteToFile("/etc/resolv.conf", buffer.String())
	if err != nil {
		err = bosherr.WrapError(err, "Writing to /etc/resolv.conf")
		return
	}

	return
}

const CENTOS_RESOLV_CONF_TEMPLATE = `{{ range .DnsServers }}nameserver {{ . }}
{{ end }}`

func (p centos) detectMacAddresses() (addresses map[string]string, err error) {
	addresses = map[string]string{}

	filePaths, err := p.linux.GetFs().Glob("/sys/class/net/*")
	if err != nil {
		err = bosherr.WrapError(err, "Getting file list from /sys/class/net")
		return
	}

	var macAddress string
	for _, filePath := range filePaths {
		macAddress, err = p.linux.GetFs().ReadFile(filepath.Join(filePath, "address"))
		if err != nil {
			err = bosherr.WrapError(err, "Reading mac address from file")
			return
		}

		interfaceName := filepath.Base(filePath)
		addresses[macAddress] = interfaceName
	}

	return
}

func (p centos) restartNetwork() {
	p.linux.GetRunner().RunCommand("service", "network", "restart")
	return
}
