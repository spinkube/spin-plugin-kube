package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func yesOrNo(question string) (bool, error) {
	fmt.Print(question)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		return true, nil
	}

	return false, nil
}
