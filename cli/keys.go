package cli

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

const (
	privateKeyFile = "5kmcli.key"
	publicKeyFile  = "5kmcli.pub"
)

func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating ed25519 key pair: %w", err)
	}
	return pub, priv, nil
}

func WriteKeyPair(dir string, pub ed25519.PublicKey, priv ed25519.PrivateKey) (pubPath, privPath string, err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", fmt.Errorf("creating output directory: %w", err)
	}

	privPath = filepath.Join(dir, privateKeyFile)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: priv.Seed(),
	})
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		return "", "", fmt.Errorf("writing private key: %w", err)
	}

	pubPath = filepath.Join(dir, publicKeyFile)
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: pub,
	})
	if err := os.WriteFile(pubPath, pubPEM, 0o644); err != nil {
		return "", "", fmt.Errorf("writing public key: %w", err)
	}

	return pubPath, privPath, nil
}

func ReadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading private key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}
	if block.Type != "ED25519 PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected PEM type %q, expected \"ED25519 PRIVATE KEY\"", block.Type)
	}
	return ed25519.NewKeyFromSeed(block.Bytes), nil
}

func ReadPublicKey(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading public key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}
	if block.Type != "ED25519 PUBLIC KEY" {
		return nil, fmt.Errorf("unexpected PEM type %q, expected \"ED25519 PUBLIC KEY\"", block.Type)
	}
	return ed25519.PublicKey(block.Bytes), nil
}
