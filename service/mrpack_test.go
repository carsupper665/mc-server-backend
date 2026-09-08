package service

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func testPackFile(name, side string) MRPackFile {
	f := MRPackFile{Path: name, Hashes: map[string]string{"sha1": fmt.Sprintf("%x", sha1.Sum([]byte("mod"))), "sha512": fmt.Sprintf("%x", sha512.Sum512([]byte("mod")))}, Downloads: []string{"https://cdn.modrinth.com/mod"}, FileSize: 3}
	if side != "" {
		f.Env = map[string]string{"server": side}
	}
	return f
}

func testPack() MRPack {
	return MRPack{FormatVersion: 1, Game: "minecraft", Name: "Test pack", VersionID: "1", Dependencies: map[string]string{"minecraft": "1.21.6", "fabric-loader": "0.18.1"}, Files: []MRPackFile{testPackFile("mods/test.jar", "")}}
}

func packBytes(t *testing.T, pack MRPack, overrides ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	f, err := z.Create("modrinth.index.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(f).Encode(pack); err != nil {
		t.Fatal(err)
	}
	for _, name := range overrides {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestMRPackFixture(t *testing.T) {
	data, err := os.ReadFile("../test/mrcpack/test_mod_pack.mrpack")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := ParseMRPack(bytes.NewReader(data), int64(len(data)), strings.Split(DefaultMRPackBlacklist, ","))
	if err != nil {
		t.Fatal(err)
	}
	if pack.Name != "EZRedstone" || pack.Dependencies["fabric-loader"] != "0.18.1" {
		t.Fatalf("wrong fixture metadata: %+v", pack)
	}
	if len(pack.Files) == 0 || len(pack.Overrides) == 0 || len(pack.Skipped) == 0 {
		t.Fatalf("missing fixture contents files=%d overrides=%d skipped=%d", len(pack.Files), len(pack.Overrides), len(pack.Skipped))
	}
	for _, file := range pack.Files {
		if file.Env["server"] == "unsupported" {
			t.Fatal("client mod selected")
		}
	}
	t.Logf("fixture: %d server files, %d overrides, %d excluded", len(pack.Files), len(pack.Overrides), len(pack.Skipped))
}

func TestMRPackEnvironmentAndLayers(t *testing.T) {
	p := testPack()
	p.Files = []MRPackFile{testPackFile("mods/required.jar", "required"), testPackFile("mods/optional.jar", "optional"), testPackFile("mods/default.jar", ""), testPackFile("mods/client.jar", "unsupported")}
	data := packBytes(t, p, "server-overrides/config/test.txt", "overrides/config/test.txt", "overrides/options.txt", "client-overrides/only.txt")
	parsed, err := ParseMRPack(bytes.NewReader(data), int64(len(data)), []string{"options.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Files) != 3 || len(parsed.Overrides) != 2 || parsed.Overrides[0].Name != "overrides/config/test.txt" || parsed.Overrides[1].Name != "server-overrides/config/test.txt" {
		t.Fatalf("wrong filtering/layers: %+v", parsed)
	}
	parsed, err = ParseMRPack(bytes.NewReader(data), int64(len(data)), nil)
	if err != nil || len(parsed.Overrides) != 3 {
		t.Fatalf("adjustable blacklist failed: %v", err)
	}
	t.Setenv("MRPACK_BLACKLIST", "")
	if packBlocked("options.txt", MRPackBlacklist()) {
		t.Fatal("empty blacklist should disable it")
	}
}

func TestMRPackRejectsInvalidInput(t *testing.T) {
	for _, name := range []string{"../escape", "/escape", "C:/escape", `mods\..\escape`} {
		t.Run(name, func(t *testing.T) {
			p := testPack()
			p.Files[0].Path = name
			data := packBytes(t, p)
			if _, err := ParseMRPack(bytes.NewReader(data), int64(len(data)), nil); err == nil {
				t.Fatal("accepted unsafe path")
			}
		})
	}
	for _, change := range []func(*MRPack){
		func(p *MRPack) { p.FormatVersion = 2 },
		func(p *MRPack) { p.Files[0].Hashes["sha512"] = "bad" },
		func(p *MRPack) { p.Files[0].Downloads = []string{"http://127.0.0.1/file"} },
		func(p *MRPack) { p.Files[0].Env = map[string]string{"server": "typo"} },
		func(p *MRPack) { p.Files = append(p.Files, p.Files[0]) },
	} {
		p := testPack()
		change(&p)
		data := packBytes(t, p)
		if _, err := ParseMRPack(bytes.NewReader(data), int64(len(data)), nil); err == nil {
			t.Fatal("accepted invalid manifest")
		}
	}
	data := packBytes(t, testPack(), "overrides/../escape")
	if _, err := ParseMRPack(bytes.NewReader(data), int64(len(data)), nil); err == nil {
		t.Fatal("accepted unsafe override")
	}
	t.Setenv("MRPACK_MAX_SIZE_MB", "1")
	data = packBytes(t, testPack(), strings.Repeat("a", 200))
	if _, err := ParseMRPack(bytes.NewReader(data), 2<<20, nil); err == nil {
		t.Fatal("accepted oversized ZIP")
	}
	p := testPack()
	p.Dependencies["future-loader"] = "1"
	if _, _, err := p.Loader(); err == nil {
		t.Fatal("unknown loader ignored")
	}
}
