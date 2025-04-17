// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"
)

const (
	reonboardPrompt         = "Looks like your machine is already onboarded. Proceed with reonboarding?"
	passphrasePrompt        = "Passphrase"
	confirmPassphrasePrompt = "Please confirm your passphrase"
)

// PromptReonboard is a wrapper of utils.PromptYesNo with custom prompt that return error if user declines reonboard
func PromptReonboard(r io.Reader, w io.Writer) error {
	confirmed, err := PromptYesNo(r, w, reonboardPrompt)
	if err != nil {
		return fmt.Errorf("could not confirm reonboarding: %w", err)
	}

	if !confirmed {
		return fmt.Errorf("reonboarding aborted by user")
	}

	return nil
}

func PromptForPassphrase(confirm bool) (string, error) {
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
		for range maxTries {
			fmt.Printf("%s: ", passphrasePrompt)
			bytePassphrase, err = term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				err = fmt.Errorf("failed to read passphrase: %w", err)
				return
			}
			fmt.Println("") // new line after passphrase input

			if confirm {
				fmt.Printf("%s: ", confirmPassphrasePrompt)
				byteConfirmation, err = term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					err = fmt.Errorf("failed to read passphrase confirmation: %w", err)
					return
				}
				fmt.Println("") // new line after passphrase confirm input

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

// PromptYesNo loops on confirmation from user until valid answer
func PromptYesNo(in io.Reader, out io.Writer, prompt string) (bool, error) {
	reader := bufio.NewReader(in)

	for {
		fmt.Fprintf(out, "%s (y/N): ", prompt)

		response, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("read response string failed: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
	}
}
