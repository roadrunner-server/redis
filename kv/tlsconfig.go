package kv

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"golang.org/x/sys/cpu"
)

type TLSConfig struct {
	Cert   string `mapstructure:"cert"`
	Key    string `mapstructure:"key"`
	RootCa string `mapstructure:"root_ca"`
}

func tlsConfig(conf *TLSConfig) (*tls.Config, error) {
	if conf == nil {
		return nil, nil
	}

	tlsConfig := defaultTLSConfig(conf)
	if conf.RootCa != "" {
		// error is always nil here
		certPool, err := x509.SystemCertPool()
		if err != nil {
			// error is always nil here
			return nil, err
		}

		if certPool == nil {
			certPool = x509.NewCertPool()
		}

		// we already checked this file in the config.go
		rca, err := os.ReadFile(conf.RootCa)
		if err != nil {
			return nil, err
		}

		if ok := certPool.AppendCertsFromPEM(rca); !ok {
			return nil, err
		}

		tlsConfig.RootCAs = certPool
	}

	if _, crtExistErr := os.Stat(conf.RootCa); crtExistErr != nil {
		return nil, crtExistErr
	}

	return tlsConfig, nil
}

func defaultTLSConfig(cfg *TLSConfig) *tls.Config {
	var topCipherSuites []uint16
	var defaultCipherSuitesTLS13 []uint16

	hasGCMAsmAMD64 := cpu.X86.HasAES && cpu.X86.HasPCLMULQDQ
	hasGCMAsmARM64 := cpu.ARM64.HasAES && cpu.ARM64.HasPMULL
	// Keep in sync with crypto/aes/cipher_s390x.go.
	hasGCMAsmS390X := cpu.S390X.HasAES && cpu.S390X.HasAESCBC && cpu.S390X.HasAESCTR && (cpu.S390X.HasGHASH || cpu.S390X.HasAESGCM)

	hasGCMAsm := hasGCMAsmAMD64 || hasGCMAsmARM64 || hasGCMAsmS390X

	if hasGCMAsm {
		// If AES-GCM hardware is provided then priorities AES-GCM
		// cipher suites.
		topCipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		}
		defaultCipherSuitesTLS13 = []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
		}
	} else {
		// Without AES-GCM hardware, we put the ChaCha20-Poly1305
		// cipher suites first.
		topCipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		}
		defaultCipherSuitesTLS13 = []uint16{
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
		}
	}

	defaultCipherSuites := make([]uint16, 0, 22)
	defaultCipherSuites = append(defaultCipherSuites, topCipherSuites...)
	defaultCipherSuites = append(defaultCipherSuites, defaultCipherSuitesTLS13...)

	return &tls.Config{
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
		},
		GetClientCertificate: getClientCertificate(cfg),
		CipherSuites:         defaultCipherSuites,
		MinVersion:           tls.VersionTLS12,
	}
}

// getClientCertificate is used for tls.Config struct field GetClientCertificate and enables re-fetching the client certificates when they expire
func getClientCertificate(cfg *TLSConfig) func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	if cfg == nil || cfg.Cert == "" || cfg.Key == "" {
		return nil
	}
	return func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
		cert, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		if err != nil {
			return nil, err
		}

		return &cert, nil
	}
}
