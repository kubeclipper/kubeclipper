package v1

import (
	"testing"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/certs"
)

func TestHasSecureWebTerminalKey(t *testing.T) {
	weakPrivate, weakPublic, err := certs.GetSSHKeyPair(1024)
	if err != nil {
		t.Fatal(err)
	}
	if hasSecureWebTerminalKey(corev1.WebTerminal{PrivateKey: weakPrivate, PublicKey: weakPublic}) {
		t.Fatal("key smaller than 2048 bits must be rejected")
	}

	privateKey, publicKey, err := certs.GetSSHKeyPair(certs.DefaultRSAKeySize)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSecureWebTerminalKey(corev1.WebTerminal{PrivateKey: privateKey, PublicKey: publicKey}) {
		t.Fatal("2048-bit web terminal key must be accepted")
	}
}
