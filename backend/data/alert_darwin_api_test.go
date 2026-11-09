//go:build darwin
// +build darwin

package data

import (
	"nws510obtl25p1Il/backend/logger"
	"testing"

	"github.com/go-toast/toast"
)

// @Author 2lovecode
// @Date 2025/02/06 17:50
// @Desc
// -----------------------------------------------------------------------------------

func TestAlert(t *testing.T) {
	notification := toast.Notification{
		AppID:    "nws510obtl25p1Il",
		Title:    "Hello, World!",
		Message:  "This is a toast notification.",
		Icon:     "../../build/appicon.png",
		Duration: "short",
		Audio:    toast.Default,
	}
	err := notification.Push()
	if err != nil {
		logger.SugaredLogger.Error(err)
		return
	}
}
