package kv

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/roadrunner-server/errors"
	"go.uber.org/zap"
	"os"
)

type TLSConfig struct {
	RootCa string `mapstructure:"root_ca"`
}

func NewTLSConfig(c *TLSConfig, log *zap.Logger) (*tls.Config, error) {
	if c == nil || c.RootCa == "" {
		return nil, nil
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	rootCAs, sysCertErr := x509.SystemCertPool()
	if sysCertErr != nil {
		rootCAs = x509.NewCertPool()
		log.Warn("unable to load system certificate pool, using empty pool", zap.Error(sysCertErr))
	}

	if _, crtExistErr := os.Stat(c.RootCa); crtExistErr != nil {
		return nil, crtExistErr
	}

	bytes, crtReadErr := os.ReadFile(c.RootCa)
	if crtReadErr != nil {
		return nil, crtReadErr
	}

	if !rootCAs.AppendCertsFromPEM(bytes) {
		return nil, errors.Errorf("failed to parse certificates from PEM file '%s'. Please ensure the file contains valid PEM-encoded certificates", c.RootCa)
	}
	tlsConfig.RootCAs = rootCAs
	return tlsConfig, nil
}
