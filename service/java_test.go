package service

import (
	"context"
	"sync/atomic"
	"testing"
)

func clearJavaEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"JAVA_8", "JAVA_17", "JAVA_21", "JAVA_25"} {
		t.Setenv(k, "")
	}
}

func TestSelectJavaBuckets(t *testing.T) {
	clearJavaEnv(t)
	t.Setenv("JAVA_8", `C:\java\8\bin\java.exe`)
	t.Setenv("JAVA_17", `C:\java\17\bin\java.exe`)
	t.Setenv("JAVA_21", `C:\java\21\bin\java.exe`)
	cases := map[int]string{
		8: `C:\java\8\bin\java.exe`, 16: `C:\java\17\bin\java.exe`, 17: `C:\java\17\bin\java.exe`,
		21: `C:\java\21\bin\java.exe`, 25: "java", 26: "java",
	}
	for major, want := range cases {
		if got := SelectJava(major); got != want {
			t.Errorf("major %d → %s, want %s", major, got, want)
		}
	}
	t.Setenv("JAVA_25", `C:\java\25\bin\java.exe`)
	if got := SelectJava(25); got != `C:\java\25\bin\java.exe` {
		t.Errorf("major 25 → %s, want JAVA_25", got)
	}
	if got := SelectJava(22); got != `C:\java\25\bin\java.exe` {
		t.Errorf("major 22 → %s, want JAVA_25", got)
	}
}

func TestSelectJavaFallsBackToPath(t *testing.T) {
	clearJavaEnv(t)
	t.Setenv("JAVA_21", `C:\java\21\bin\java.exe`)
	if got := SelectJava(8); got != "java" {
		t.Fatalf("missing JAVA_8 must use PATH java, got %s", got)
	}
	if got := SelectJava(17); got != "java" {
		t.Fatalf("missing JAVA_17 must use PATH java, got %s", got)
	}
}

func TestJavaForMCUsesMojangData(t *testing.T) {
	var calls atomic.Int32
	setupVersionTransport(t, &calls)
	clearJavaEnv(t)
	if got := JavaForMC(context.Background(), "1.16.5"); got != "java" || calls.Load() != 0 {
		t.Fatalf("no JAVA_* configured must not fetch version data: %s calls=%d", got, calls.Load())
	}
	t.Setenv("JAVA_8", "/j8")
	t.Setenv("JAVA_17", "/j17")
	t.Setenv("JAVA_21", "/j21")
	for mc, want := range map[string]string{"1.12.2": "/j8", "1.16.5": "/j8", "1.20.1": "/j17", "1.21.4": "/j21"} {
		if got := JavaForMC(context.Background(), mc); got != want {
			t.Errorf("%s → %s, want %s", mc, got, want)
		}
	}
	if calls.Load() == 0 {
		t.Fatal("Mojang version data not used")
	}
}

func TestFallbackJavaMajor(t *testing.T) {
	for mc, want := range map[string]int{"1.12.2": 8, "1.16.5": 8, "1.17.1": 16, "1.18.2": 17, "1.20.4": 17, "1.20.5": 21, "1.21.4": 21, "26.1": 25} {
		if got := FallbackJavaMajor(mc); got != want {
			t.Errorf("%s → %d, want %d", mc, got, want)
		}
	}
}
