//go:build kc_default

package k8s

var nodeScript = `
systemctl stop firewalld || true
systemctl disable firewalld || true
setenforce 0
sed -i s/^SELINUX=.*$/SELINUX=disabled/ /etc/selinux/config
modprobe br_netfilter && modprobe nf_conntrack
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward=1
EOF
cat > /etc/sysctl.conf << EOF
net.ipv6.conf.all.forwarding=1
fs.file-max = 100000
vm.max_map_count=262144
EOF
cat > /etc/security/limits.conf << EOF
#IncreaseMaximumNumberOfFileDescriptors
* soft nproc 65535
* hard nproc 65535
* soft nofile 65535
* hard nofile 65535
#IncreaseMaximumNumberOfFileDescriptors
EOF
sysctl --system
sysctl -p
swapoff -a
sed -i /swap/d /etc/fstab`
