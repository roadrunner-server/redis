package kv

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/roadrunner-server/errors"
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
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}

		if certPool == nil {
			certPool = x509.NewCertPool()
		}

		rca, err := os.ReadFile(conf.RootCa)
		if err != nil {
			return nil, err
		}

		if ok := certPool.AppendCertsFromPEM(rca); !ok {
			return nil, errors.Str("failed to append certificates from PEM")
		}

		tlsConfig.RootCAs = certPool
	}

	return tlsConfig, nil
}

func defaultTLSConfig(cfg *TLSConfig) *tls.Config {
	return &tls.Config{
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
		},
		GetClientCertificate: getClientCertificate(cfg),
		MinVersion:           tls.VersionTLS12,
	}
}

// getClientCertificate is used for tls.Config struct field GetClientCertificate and enables re-fetching the client certificates when they expire
func getClientCertificate(cfg *TLSConfig) func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	if cfg == nil || (cfg.Cert == "" && cfg.Key == "") {
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
