package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func SendRequestCreateProfile(userID uuid.UUID) error {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	payload := map[string]interface{}{"user_id": userID.String()}

	body, _ := json.Marshal(payload)

	resp, err := httpClient.Post(
		"http://user:8002/api/user/profile",
		"application/json",
		bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("users service returned status: %d", resp.StatusCode)
	}

	return nil
}
