// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cert

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

const (
	PEMBlockRSAPrivateKey = "RSA PRIVATE KEY"
	PEMBlockECPrivateKey  = "EC PRIVATE KEY"
	PEMBlockCertificate   = "CERTIFICATE"
)

func publicKey(priv any) any {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv any) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: PEMBlockRSAPrivateKey, Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: PEMBlockECPrivateKey, Bytes: b}
	default:
		return nil
	}
}

// DERBytesToPEMBlocks converts certificate DER bytes and optional private key
// to PEM blocks.
// Returns certificate PEM block and private key PEM block.
func DERBytesToPEMBlocks(der []byte, priv any) ([]byte, []byte, error) {
	out := &bytes.Buffer{}
	if err := pem.Encode(out, &pem.Block{Type: PEMBlockCertificate, Bytes: der}); err != nil {
		return nil, nil, err
	}
	cert := append([]byte{}, out.Bytes()...)

	var key []byte
	if priv != nil {
		out.Reset()
		if err := pem.Encode(out, pemBlockForKey(priv)); err != nil {
			return nil, nil, err
		}
		key = append([]byte{}, out.Bytes()...)
	}

	return cert, key, nil
}
