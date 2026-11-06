package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/alltomatos/dontstop/gateway/internal/platform/config"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.MasterKey == "" {
		return fmt.Errorf("MASTER_KEY is required")
	}
	box, err := secret.NewBoxFromBase64(cfg.MasterKey)
	if err != nil {
		return err
	}
	plaintext, err := io.ReadAll(io.LimitReader(os.Stdin, 16<<10))
	if err != nil {
		return fmt.Errorf("read plaintext: %w", err)
	}
	if len(plaintext) == 0 {
		return fmt.Errorf("plaintext is empty")
	}
	ciphertext, err := box.Encrypt(plaintext)
	if err != nil {
		return err
	}
	fmt.Print(base64.StdEncoding.EncodeToString(ciphertext))
	return nil
}
