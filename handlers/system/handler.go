package system

import (
	"context"
	"net/http"
)

type SystemConfigRequest struct{}

type VersionResponse struct {
	MinVersion      string `json:"min_version"`
	LatestVersion   string `json:"latest_version"`
	ForceUpdate     bool   `json:"force_update"`
	StoreUrlAndroid string `json:"store_url_android"`
	StoreUrlIOS     string `json:"store_url_ios"`
	Message         string `json:"message"`
}

type SystemConfigHandler struct{}

// NewSystemConfigHandler creates a new SystemConfigHandler instance.
//
//	@Summary		System Config
//	@Description	Get system configuration like app version
//	@Tags			System
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	VersionResponse
//	@Failure		500
//	@Router			/system/config [get]
func NewSystemConfigHandler() *SystemConfigHandler {
	return &SystemConfigHandler{}
}

func (h *SystemConfigHandler) Handle(ctx context.Context, req *SystemConfigRequest) (*VersionResponse, int, error) {
	config := &VersionResponse{
		MinVersion:      "1.0.5",
		LatestVersion:   "1.0.5",
		ForceUpdate:     false,
		StoreUrlAndroid: "market://details?id=com.kuarulabs.scope",
		StoreUrlIOS:     "https://apps.apple.com/app/idSENIN_APP_ID",
		Message:         "Sana daha iyi bir deneyim sunmak için uygulamanın altyapısını güncelledik. Devam etmek için lütfen uygulamayı güncelle! 🚀",
	}

	return config, http.StatusOK, nil
}
