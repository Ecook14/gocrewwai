package utils_test

import (
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

func TestValidateURL_BlocksSSRF(t *testing.T) {
	t.Setenv("GOCREW_ALLOW_PRIVATE_URLS", "")
	for _, raw := range []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://localhost:8080/admin",
		"http://127.0.0.1:9000/",
		"http://[::1]/",
		"http://192.168.1.1/",
		"http://10.0.0.5/",
		"ftp://example.com/x",
		"http://user:pass@example.com/",
		"http:///no-host",
	} {
		if _, err := utils.ValidateURL(raw); err == nil {
			t.Errorf("expected block for %q", raw)
		}
	}
}

func TestValidateURL_AllowsPublic(t *testing.T) {
	t.Setenv("GOCREW_ALLOW_PRIVATE_URLS", "")
	if _, err := utils.ValidateURL("https://example.com/api"); err != nil {
		t.Errorf("expected allow for public URL: %v", err)
	}
}

func TestValidateURL_PrivateAllowedWithEnv(t *testing.T) {
	t.Setenv("GOCREW_ALLOW_PRIVATE_URLS", "1")
	if _, err := utils.ValidateURL("http://127.0.0.1:8080/"); err != nil {
		t.Errorf("expected allow with env override: %v", err)
	}
}

func TestValidatePath_SiblingPrefix(t *testing.T) {
	if _, err := utils.ValidatePath("/dataX/foo", "/data"); err == nil {
		t.Error("expected rejection of sibling-prefix path")
	}
	if _, err := utils.ValidatePath("/data/../etc/passwd", "/data"); err == nil {
		t.Error("expected rejection of traversal")
	}
	root := t.TempDir()
	if _, err := utils.ValidatePathResolved(root+"/file.txt", root); err != nil {
		t.Errorf("expected accept for in-root path: %v", err)
	}
	if _, err := utils.ValidatePathResolved(root+"/../escape.txt", root); err == nil {
		t.Error("expected rejection of resolved traversal")
	}
}
