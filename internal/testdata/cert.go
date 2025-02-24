package testdata

import (
	"crypto/tls"
	"crypto/x509"
	"path"
	"runtime"
)

var certPath string

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current frame")
	}

	certPath = path.Dir(filename)
}

// GetCertificatePaths returns the paths to certificate and key
func GetCertificatePaths() (string, string) {
	return path.Join(certPath, "cac.gachenot.eu.crt"), path.Join(certPath, "cac.gachenot.eu.key")
}

// GetTLSConfig returns a tls config for quic.clemente.io
func GetTLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair(GetCertificatePaths())
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
	}
}

// AddRootCA adds the root CA certificate to a cert pool
func AddRootCA(certPool *x509.CertPool) {
	caCert := []byte(`-----BEGIN CERTIFICATE-----
MIID6zCCAtOgAwIBAgIUaWdq8+0Qk9qE1qK2toHvYnETe1owDQYJKoZIhvcNAQEL
BQAwgYQxCzAJBgNVBAYTAkZSMQ0wCwYDVQQIDAROb3JkMQ4wDAYDVQQHDAVMaWxs
ZTEbMBkGA1UECgwSR2FjaGVub3QgU29mdHdhcmVzMRgwFgYDVQQDDA9jYWMuZ2Fj
aGVub3QuZXUxHzAdBgkqhkiG9w0BCQEWEGJnMzMzQGtlbnQuYWMudWswHhcNMjMw
ODAzMTEwNDE1WhcNMjgwODAxMTEwNDE1WjCBhDELMAkGA1UEBhMCRlIxDTALBgNV
BAgMBE5vcmQxDjAMBgNVBAcMBUxpbGxlMRswGQYDVQQKDBJHYWNoZW5vdCBTb2Z0
d2FyZXMxGDAWBgNVBAMMD2NhYy5nYWNoZW5vdC5ldTEfMB0GCSqGSIb3DQEJARYQ
YmczMzNAa2VudC5hYy51azCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AI8u7T/UHDPiIxc2BUhw3d8XSGcK4iN1g7DnfvNWhNmCSypgZEoVmGYUPXQ8q5uw
VMO92JJcsSZCRVsWG3D28eHvq7Rp9Mow9Lj0ST++2XoNNjjWH/75VXjkJVFgYk9h
h0TZjrRjCC/ZFpDYv5CsxSseivmz5Z34t3JBPvkmYHwpY9UPlniZEuzrAPSrUiwT
Jx9CQOVjpUsFR+UxR4aXlXJf0cW+SOP2QJujocim9FqwBfQIZpwF7cwlHuk78kLT
GH3KqB2gT+2WLDkoJRKlP2ay6m4tR57RFYNeEFJcn9CpnQxPB7KDiQij322lv58e
2Me9EC4mrk7VKVJ6uQiJ100CAwEAAaNTMFEwHQYDVR0OBBYEFJBWG90p2ajo8FOd
oiRgxrBOHB6+MB8GA1UdIwQYMBaAFJBWG90p2ajo8FOdoiRgxrBOHB6+MA8GA1Ud
EwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAGijgh1yJR9GXHLPmQFLuc5D
q1thbLni9EBxEZmpIGnMpFrpFt9hb0ctAirqRMOmAY1TpMQ2BxtHxlz5wYu8LZ5I
uyL6hu17IqFRH9t/tWIYZjI3FuRuRVm5tm9egVmOV++35ngxyg+OeewKNeZYCR+j
gjGGLiP3fg1wVQhH8RHZT7ppPZUSHnbWvGIcqp6yylTdc31IlQFU09WhIOdyxIhI
IoEUzV/qYD9YWv4zqUgmcF/gIoN1oA33jMT3GIRsSHB5vkcDwPFZOSuDwfms7J+V
Rt8HNFerFBJBTDmYYdyp/JSLHVHb2kYNC0dcrLXeYLiHPznlhgro+XgK7QCPdOo=
-----END CERTIFICATE-----`)
	if ok := certPool.AppendCertsFromPEM(caCert); !ok {
		panic("Could not add root ceritificate to pool.")
	}
}

// GetRootCA returns an x509.CertPool containing (only) the CA certificate
func GetRootCA() *x509.CertPool {
	pool := x509.NewCertPool()
	AddRootCA(pool)
	return pool
}
