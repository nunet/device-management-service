package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/howeyc/gopass"

	"gitlab.com/nunet/device-management-service/utils"
)

// promptReonboard is a wrapper of utils.PromptYesNo with custom prompt that return error if user declines reonboard
func promptReonboard(r io.Reader, w io.Writer) error {
	prompt := "Looks like your machine is already onboarded. Proceed with reonboarding?"

	confirmed, err := utils.PromptYesNo(r, w, prompt)
	if err != nil {
		return fmt.Errorf("could not confirm reonboarding: %w", err)
	}

	if !confirmed {
		return fmt.Errorf("reonboarding aborted by user")
	}

	return nil
}

func promptForPassphrase(confirm bool) (string, error) {
	maxTries := 3

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	done := make(chan bool)

	var passphrase string
	var err error

	// Start a goroutine to handle passphrase input
	go func() {
		defer close(done)
		var bytePassphrase, byteConfirmation []byte
		for i := 0; i < maxTries; i++ {
			fmt.Print("Passphrase: ")
			bytePassphrase, err = gopass.GetPasswdMasked()
			if err != nil {
				err = fmt.Errorf("failed to read passphrase: %w", err)
				return
			}

			if confirm {
				fmt.Print("Please confirm your passphrase: ")
				byteConfirmation, err = gopass.GetPasswdMasked()
				if err != nil {
					err = fmt.Errorf("failed to read passphrase confirmation: %w", err)
					return
				}

				if string(bytePassphrase) != string(byteConfirmation) {
					err = fmt.Errorf("passphrases do not match")
				}
			}
			if err == nil {
				passphrase = string(bytePassphrase)
				return
			}
			fmt.Println(err)
			fmt.Println("")
		}

		err = fmt.Errorf("user failed to input passphrase")
	}()

	// Wait for either the passphrase input to complete or an interrupt signal
	select {
	case <-done:
		return passphrase, err
	case <-sigChan:
		return "", errors.New("sigterm received")
	}
}
