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
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/certs"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

var _ = SIGDescribe("[Fast] [Serial] Node terminal connect", func() {
	f := framework.NewDefaultFramework("node")
	msg := ""
	nodeID := ""
	credential := &v1.SSHCredential{}

	ginkgo.BeforeEach(func() {
		ginkgo.By("Check that there are enough available nodes")
		nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
			Pagination: query.NoPagination(),
		})
		framework.ExpectNoError(err)
		nodeID = nodes.Items[0].Name

		ginkgo.By("Get public key")
		pub, err := f.Client.GetPublicKey(context.TODO())
		framework.ExpectNoError(err)

		ginkgo.By("Get msg")
		pubkey, _ := base64.StdEncoding.DecodeString(pub.PublicKey)
		u, err := certs.RsaEncrypt([]byte("root"), pubkey)
		framework.ExpectNoError(err)
		p, err := certs.RsaEncrypt([]byte("Thinkbig1"), pubkey)
		framework.ExpectNoError(err)

		credential = &v1.SSHCredential{
			Username: u,
			Password: p,
			Port:     22,
		}
		jsonBody, err := json.Marshal(credential)
		msg = base64.StdEncoding.EncodeToString(jsonBody)
		framework.ExpectNoError(err)
	})

	ginkgo.It("connect the node and ensure node is connected", func() {
		ginkgo.By("connect node")
		url := fmt.Sprintf("ws://%s%s/%s/%sname=%s&token=%s&msg=%s", f.Client.Host(), kc.ListNodesPath, nodeID, "terminal?", nodeID, f.Client.BearerToken, msg)
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
})
