package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func main() {
	outputDir := flag.String("out", "./certs", "Output directory for certificate files")
	org := flag.String("org", "LogHub", "Organization name for the certificate")
	cn := flag.String("cn", "LogHub WSS Server", "Common name for the certificate")
	validDays := flag.Int("days", 365, "Certificate validity in days")
	hosts := flag.String("hosts", "localhost,127.0.0.1", "Comma-separated list of hostnames and IPs")
	flag.Parse()

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== LogHub Certificate Generator ===")
	fmt.Printf("Organization: %s\n", *org)
	fmt.Printf("Common Name:  %s\n", *cn)
	fmt.Printf("Valid Days:   %d\n", *validDays)
	fmt.Printf("Hosts:        %s\n", *hosts)
	fmt.Printf("Output Dir:   %s\n", *outputDir)
	fmt.Println()

	// Generate CA certificate
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating CA key: %v\n", err)
		os.Exit(1)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{*org},
			CommonName:   *org + " CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CA certificate: %v\n", err)
		os.Exit(1)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing CA certificate: %v\n", err)
		os.Exit(1)
	}

	// Generate server certificate
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating server key: %v\n", err)
		os.Exit(1)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{*org},
			CommonName:   *cn,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(0, 0, *validDays),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	// Parse hosts
	hostList := splitHosts(*hosts)
	for _, h := range hostList {
		if ip := net.ParseIP(h); ip != nil {
			serverTemplate.IPAddresses = append(serverTemplate.IPAddresses, ip)
		} else {
			serverTemplate.DNSNames = append(serverTemplate.DNSNames, h)
		}
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating server certificate: %v\n", err)
		os.Exit(1)
	}

	// Write CA certificate
	caFile := filepath.Join(*outputDir, "ca.crt")
	if err := writePEM(caFile, "CERTIFICATE", caCertDER); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing CA cert: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ CA certificate:     %s\n", caFile)

	// Write CA key
	caKeyFile := filepath.Join(*outputDir, "ca.key")
	caKeyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling CA key: %v\n", err)
		os.Exit(1)
	}
	if err := writePEM(caKeyFile, "EC PRIVATE KEY", caKeyDER); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing CA key: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ CA private key:     %s\n", caKeyFile)

	// Write server certificate
	serverCertFile := filepath.Join(*outputDir, "server.crt")
	if err := writePEM(serverCertFile, "CERTIFICATE", serverCertDER); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing server cert: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Server certificate: %s\n", serverCertFile)

	// Write server key
	serverKeyFile := filepath.Join(*outputDir, "server.key")
	serverKeyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling server key: %v\n", err)
		os.Exit(1)
	}
	if err := writePEM(serverKeyFile, "EC PRIVATE KEY", serverKeyDER); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing server key: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Server private key: %s\n", serverKeyFile)

	fmt.Println()
	fmt.Println("Certificate generation completed successfully!")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  Server cert: %s\n", serverCertFile)
	fmt.Printf("  Server key:  %s\n", serverKeyFile)
	fmt.Printf("  CA cert:     %s (for client trust)\n", caFile)
}

func writePEM(filename, blockType string, data []byte) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return pem.Encode(f, &pem.Block{
		Type:  blockType,
		Bytes: data,
	})
}

func splitHosts(hosts string) []string {
	var result []string
	current := ""
	for _, c := range hosts {
		if c == ',' {
			if s := trim(current); s != "" {
				result = append(result, s)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if s := trim(current); s != "" {
		result = append(result, s)
	}
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
