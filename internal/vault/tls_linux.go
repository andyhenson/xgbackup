//go:build linux

package vault

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/xgbackup/xgbackup/internal/config"
)

func webCertPaths(w config.WebConfig) (string, string) {
	cert, key := w.Cert, w.Key
	if cert == "" {
		cert = "/etc/xgbackup/tls/cert.pem"
	}
	if key == "" {
		key = "/etc/xgbackup/tls/key.pem"
	}
	return cert, key
}

func ensureWebCert(certPath, keyPath string) error {
	if st, err := os.Stat(certPath); err == nil && !st.IsDir() {
		if st, err := os.Stat(keyPath); err == nil && !st.IsDir() {
			return nil
		}
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "xgbackup"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	ifaces, _ := net.InterfaceAddrs()
	for _, addr := range ifaces {
		ipnet, ok := addr.(*net.IPNet)
		if ok && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ipnet.IP)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0o755); err != nil {
		return err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return err
	}
	return os.WriteFile(keyPath, keyPEM, 0o600)
}
