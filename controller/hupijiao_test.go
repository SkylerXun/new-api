package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestHupijiaoSign(t *testing.T) {
	values := map[string]string{"b": "2", "a": "1", "empty": "", "hash": "old"}
	require.Equal(t, "8d9f51949e440aa629fd1a035708473a", hupijiaoSign(values, "secret"))
	if hupijiaoSign(map[string]string{"a": "1"}, "secret") == hupijiaoSign(map[string]string{"a": "1", "hash": "bad"}, "secret") {
		t.Log("hash field excluded")
	} else {
		require.Fail(t, "hash field must be excluded")
	}
}

func TestHupijiaoRequestFallsBackToBackup(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer primary.Close()
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payment/do.html", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "app", r.Form.Get("appid"))
		require.Equal(t, hupijiaoSign(map[string]string{"appid": "app"}, "secret"), r.Form.Get("hash"))
		_, _ = w.Write([]byte(`{"errcode":0,"url":"https://pay.example/h5","url_qrcode":"qr-content"}`))
	}))
	defer backup.Close()

	originalPrimary, originalBackup := operation_setting.HupijiaoAPIAddress, operation_setting.HupijiaoBackupAPIAddress
	operation_setting.HupijiaoAPIAddress, operation_setting.HupijiaoBackupAPIAddress = primary.URL, backup.URL
	t.Cleanup(func() {
		operation_setting.HupijiaoAPIAddress, operation_setting.HupijiaoBackupAPIAddress = originalPrimary, originalBackup
	})

	result, err := hupijiaoRequest(map[string]string{"appid": "app"}, "secret")
	require.NoError(t, err)
	require.Equal(t, "https://pay.example/h5", result["url"])
	require.Equal(t, "qr-content", result["url_qrcode"])
}

func TestParseHupijiaoPackagesRejectsInvalidDiscount(t *testing.T) {
	original := operation_setting.HupijiaoPackages
	operation_setting.HupijiaoPackages = `[{"id":"bad","title":"bad","original_amount":10,"quota":1000,"discount_rate":1.01,"enabled":true}]`
	t.Cleanup(func() { operation_setting.HupijiaoPackages = original })
	_, err := parseHupijiaoPackages()
	require.Error(t, err)
}
