package main

import (
	"crypto/x509"
	"net"
	"testing"
	"time"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("generateSelfSignedCert() error = %v", err)
	}

	if len(cert.Certificate) == 0 {
		t.Fatal("certificate chain is empty")
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("failed to parse leaf certificate: %v", err)
	}

	// Check subject
	if leaf.Subject.CommonName != "CurlDrop" {
		t.Errorf("CommonName = %q, want %q", leaf.Subject.CommonName, "CurlDrop")
	}

	// Check DNS SANs
	foundLocalhost := false
	for _, dns := range leaf.DNSNames {
		if dns == "localhost" {
			foundLocalhost = true
		}
	}
	if !foundLocalhost {
		t.Error("DNSNames should contain 'localhost'")
	}

	// Check IP SANs
	found127 := false
	for _, ip := range leaf.IPAddresses {
		if ip.Equal(net.ParseIP("127.0.0.1")) {
			found127 = true
		}
	}
	if !found127 {
		t.Error("IPAddresses should contain 127.0.0.1")
	}

	// Check validity period (~1 year)
	expectedExpiry := time.Now().Add(365 * 24 * time.Hour)
	diff := leaf.NotAfter.Sub(expectedExpiry)
	if diff < -24*time.Hour || diff > 24*time.Hour {
		t.Errorf("NotAfter = %v, expected ~%v", leaf.NotAfter, expectedExpiry)
	}

	// Check key usage
	if leaf.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("KeyUsage should include DigitalSignature")
	}

	// Check extended key usage
	foundServerAuth := false
	for _, eku := range leaf.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			foundServerAuth = true
		}
	}
	if !foundServerAuth {
		t.Error("ExtKeyUsage should include ServerAuth")
	}
}
