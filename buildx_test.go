package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdSetupBuildx_ForwardsStandardProxyVars(t *testing.T) {
	unsetProxyEnv(t)

	t.Setenv("http_proxy", "http://proxy.example:8080")
	t.Setenv("https_proxy", "http://proxy.example:8443")
	t.Setenv("no_proxy", "localhost")

	cmd := cmdSetupBuildx(Builder{Driver: dockerContainerDriver, Name: "test"}, nil, false)
	args := cmd.Args

	assertHasDriverOpt(t, args, "env.http_proxy=http://proxy.example:8080")
	assertHasDriverOpt(t, args, "env.https_proxy=http://proxy.example:8443")
	assertHasDriverOpt(t, args, "env.no_proxy=localhost")
	assertHasDriverOpt(t, args, "network=host")
}

func TestCmdSetupBuildx_PrefersStandardProxyOverHarness(t *testing.T) {
	unsetProxyEnv(t)

	t.Setenv("http_proxy", "http://standard:8080")
	t.Setenv("HARNESS_HTTP_PROXY", "http://harness:8080")
	t.Setenv("https_proxy", "http://standard:8443")
	t.Setenv("HARNESS_HTTPS_PROXY", "http://harness:8443")

	cmd := cmdSetupBuildx(Builder{Driver: dockerContainerDriver}, nil, false)
	args := cmd.Args

	assertHasDriverOpt(t, args, "env.http_proxy=http://standard:8080")
	assertHasDriverOpt(t, args, "env.https_proxy=http://standard:8443")
	assertNotHasDriverOpt(t, args, "env.http_proxy=http://harness:8080")
}

func TestCmdSetupBuildx_FallsBackToHarnessProxy(t *testing.T) {
	unsetProxyEnv(t)

	t.Setenv("HARNESS_HTTP_PROXY", "http://harness:8080")
	t.Setenv("HARNESS_HTTPS_PROXY", "http://harness:8443")
	t.Setenv("HARNESS_NO_PROXY", "localhost")

	cmd := cmdSetupBuildx(Builder{Driver: dockerContainerDriver}, nil, false)
	args := cmd.Args

	assertHasDriverOpt(t, args, "env.http_proxy=http://harness:8080")
	assertHasDriverOpt(t, args, "env.https_proxy=http://harness:8443")
	assertHasDriverOpt(t, args, "env.no_proxy=localhost")
	assertHasDriverOpt(t, args, "network=host")
}

func TestCmdSetupBuildx_QuotesNoProxyWhenContainsCommas(t *testing.T) {
	unsetProxyEnv(t)

	t.Setenv("http_proxy", "http://proxy.example:8080")
	t.Setenv("no_proxy", "localhost,127.0.0.1,.example.com")

	cmd := cmdSetupBuildx(Builder{Driver: dockerContainerDriver}, nil, false)
	args := cmd.Args

	assertHasDriverOpt(t, args, `"env.no_proxy=localhost,127.0.0.1,.example.com"`)
}

func TestCmdSetupBuildx_ForwardsHarnessCAPathAndContent(t *testing.T) {
	unsetProxyEnv(t)
	t.Setenv("HARNESS_CA_PATH", "")

	dir := t.TempDir()
	caPath := filepath.Join(dir, "harness-ca.crt")
	caContent := "-----BEGIN CERTIFICATE-----\nTESTCERT\n-----END CERTIFICATE-----\n"
	if err := os.WriteFile(caPath, []byte(caContent), 0644); err != nil {
		t.Fatalf("write ca file: %v", err)
	}
	t.Setenv("HARNESS_CA_PATH", caPath)

	cmd := cmdSetupBuildx(Builder{Driver: dockerContainerDriver}, nil, false)
	args := cmd.Args

	assertHasDriverOpt(t, args, "env.HARNESS_CA_PATH="+caPath)
	assertHasDriverOpt(t, args, "env.HARNESS_CA_CERT="+caContent)
}

func TestCmdSetupBuildx_OmitsCAWhenUnset(t *testing.T) {
	unsetProxyEnv(t)
	os.Unsetenv("HARNESS_CA_PATH")

	cmd := cmdSetupBuildx(Builder{Driver: dockerContainerDriver}, nil, false)
	args := cmd.Args

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--driver-opt" && strings.HasPrefix(args[i+1], "env.HARNESS_CA_") {
			t.Fatalf("unexpected CA driver opt %q", args[i+1])
		}
	}
}

func unsetProxyEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"http_proxy", "HTTP_PROXY", "HARNESS_HTTP_PROXY",
		"https_proxy", "HTTPS_PROXY", "HARNESS_HTTPS_PROXY",
		"no_proxy", "NO_PROXY", "HARNESS_NO_PROXY",
		"HARNESS_CA_PATH",
	}
	for _, key := range keys {
		os.Unsetenv(key)
	}
}

func assertHasDriverOpt(t *testing.T, args []string, want string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--driver-opt" && args[i+1] == want {
			return
		}
	}
	t.Fatalf("missing --driver-opt %q in args: %v", want, args)
}

func assertNotHasDriverOpt(t *testing.T, args []string, unwanted string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--driver-opt" && args[i+1] == unwanted {
			t.Fatalf("unexpected --driver-opt %q in args: %v", unwanted, args)
		}
	}
}
