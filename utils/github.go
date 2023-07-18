package utils

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
)

const (
	port         = "6969"
	secretEnvVar = "GITHUB_WEBHOOK_SECRET"
)

func StartWebhookReceiver() {
	http.HandleFunc("/webhook", handleWebhook)
	fmt.Printf("GitHub webhook receiver is listening on port %s...\n", port)
	http.ListenAndServe(":"+port, nil)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	secret := os.Getenv(secretEnvVar)
	if secret == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	actualSecret := r.Header.Get("X-Hub-Signature")
	if actualSecret != secret {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := gitPull()
	if err != nil {
		http.Error(w, "Error updating repository", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Webhook received and repository updated successfully!")
}

func gitPull() error {
	cmd := exec.Command("git", "pull", "-f")
	return cmd.Run()
}
