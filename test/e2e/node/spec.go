package node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/onsi/ginkgo"

	v1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/certs"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

var _ = SIGDescribe("[Serial]", func() {
	f := framework.NewDefaultFramework("node")
	var nodeID, nodeIP string

	ginkgo.BeforeEach(func() {
		ginkgo.By("Check that there are enough available nodes")
		nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
			Pagination:    query.NoPagination(),
			LabelSelector: fmt.Sprintf("!%s", common.LabelNodeDisable),
		})
		framework.ExpectNoError(err)
		nodeID = nodes.Items[0].Name
		nodeIP = nodes.Items[0].Status.Ipv4DefaultIP
	})

	ginkgo.It("[Fast] [new-node] [Enable] [Disable] should disable and enable node of kubeclipper platform", func() {
		ginkgo.By("disable node")
		err := f.Client.DisableNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("check node is disabled")
		nodeList, err := f.Client.DescribeNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)
		if _, ok := nodeList.Items[0].Labels[common.LabelNodeDisable]; !ok {
			framework.Failf("Fail to disable node")
		} else {
			ginkgo.By("node is disabled")
		}

		ginkgo.By("enable node")
		err = f.Client.EnableNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("check node is enabled")
		nodeList, err = f.Client.DescribeNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)
		if _, ok := nodeList.Items[0].Labels[common.LabelNodeDisable]; !ok {
			ginkgo.By("node is enabled")
		} else {
			framework.Failf("Fail to enabled node")
		}
	})
	ginkgo.It("[Fast] [new-node] [Terminal] should connect node ssh", func() {
		ginkgo.By("Get public key")
		pub, err := f.Client.GetPublicKey(context.TODO())
		framework.ExpectNoError(err)

		ginkgo.By("Get msg")
		pubkey, _ := base64.StdEncoding.DecodeString(pub.PublicKey)
		u, err := certs.RsaEncrypt([]byte("root"), pubkey)
		framework.ExpectNoError(err)
		p, err := certs.RsaEncrypt([]byte("Thinkbig1"), pubkey)
		framework.ExpectNoError(err)
		credential := &v1.SSHCredential{
			Username: u,
			Password: p,
			Port:     22,
		}
		jsonBody, err := json.Marshal(credential)
		msg := base64.StdEncoding.EncodeToString(jsonBody)
		framework.ExpectNoError(err)

		ginkgo.By("connect node")
		url := fmt.Sprintf("ws://%s%s/%s/%sname=%s&token=%s&msg=%s", f.Client.Host(), kc.ListNodesPath, nodeID, "terminal?", nodeID, f.Client.Token(), msg)
		ws, _, err := websocket.DefaultDialer.Dial(url, nil)
		framework.ExpectNoError(err)

		ginkgo.By("check the node is connected")
		for {
			_, data, err := ws.ReadMessage()
			if string(data) == "" || err != nil {
				framework.Failf("Fail to connect node")
			} else {
				break
			}
		}

		ginkgo.By("close the connection")
		err = ws.Close()
		framework.ExpectNoError(err)
	})
	ginkgo.It("[Fast] [new-node] [Detail] [Info] should get info and detail of node", func() {
		ginkgo.By("show node info")
		nodeList, err := f.Client.ListNodes(context.TODO(), kc.Queries{
			Pagination: query.NoPagination(),
			FuzzySearch: map[string]string{
				"default-ip": nodeIP,
			},
		})
		framework.ExpectNoError(err)
		if len(nodeList.Items) == 0 {
			framework.Failf("show node info e2e test failed, no such node")
		}

		ginkgo.By("show node detail")
		nodeList, err = f.Client.DescribeNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)
		if len(nodeList.Items) == 0 {
			framework.Failf("show node detail e2e test failed, no such node")
		}
	})
})
