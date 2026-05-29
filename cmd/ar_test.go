// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/internal/archivecli"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
)

func TestArFileIndexCommands(t *testing.T) {
	tmpDir := t.TempDir()
	fake7z := writeFake7z(t, tmpDir)

	galleryRoot := filepath.Join(tmpDir, "gallery")
	archiveRoot := filepath.Join(tmpDir, "archive")
	archiveTempRoot := filepath.Join(tmpDir, "archive-temp")
	indexRoot := filepath.Join(tmpDir, "index")
	backupRoot := filepath.Join(tmpDir, "backup")
	indexBackend := "ar-index-file"
	backupBackend := "ar-backup-file"
	configPath := writeArConfig(
		t,
		tmpDir,
		fake7z,
		galleryRoot,
		archiveRoot,
		archiveTempRoot,
		indexBackend,
		indexRoot,
		backupBackend,
		backupRoot,
		"file",
		"",
	)

	srcDir1 := filepath.Join(tmpDir, "src-101")
	srcDir2 := filepath.Join(tmpDir, "src-102")
	writeFixtureFile(t, filepath.Join(srcDir1, "a.txt"), "alpha")
	writeFixtureFile(t, filepath.Join(srcDir1, "nested", "b.txt"), "bravo")
	writeFixtureFile(t, filepath.Join(srcDir2, "c.txt"), "charlie")

	executeRoot(t, "--config", configPath, "ar", "pack", "--cid", "101", "--src-dir", srcDir1)
	executeRoot(t, "--config", configPath, "ar", "pack", "--cid", "102", "--src-dir", srcDir2)

	meta101, err := manager.Get().Get(context.Background(), 101)
	if err != nil {
		t.Fatalf("get meta 101 failed: %v", err)
	}
	if !strings.HasSuffix(meta101.Path, filepath.Join("00", "01", "101.cocoma")) {
		t.Fatalf("unexpected archive path: %s", meta101.Path)
	}
	if _, err := os.Stat(meta101.Path); err != nil {
		t.Fatalf("archive file not found: %v", err)
	}

	queryOutput := executeRoot(t, "--config", configPath, "ar", "--output", "json", "query", "--cid", "101")
	queryEnvelope := decodeEnvelope(t, queryOutput)
	queryMeta := decodeArchiveMeta(t, queryEnvelope["data"])
	if queryMeta.ID != 101 || queryMeta.Path != meta101.Path {
		t.Fatalf("query result mismatch: %+v", queryMeta)
	}

	executeRoot(t, "--config", configPath, "ar", "backup", "--cid", "101", "--backend", backupBackend, "--prefix", "replicas")
	backupStorage, ok := storage.Get(backupBackend)
	if !ok {
		t.Fatalf("backup backend not found")
	}
	if _, err := backupStorage.Stat(context.Background(), "/replicas/101.cocoma"); err != nil {
		t.Fatalf("backup object not found: %v", err)
	}
	if _, err := backupStorage.Stat(context.Background(), "/replicas/102.cocoma"); err == nil {
		t.Fatalf("unexpected backup for cid 102")
	}

	executeRoot(t, "--config", configPath, "ar", "check", "--cid", "101")
	checkedMeta, err := manager.Get().Get(context.Background(), 101)
	if err != nil {
		t.Fatalf("get checked meta failed: %v", err)
	}
	if !checkedMeta.ReplicaHealth.Healthy {
		t.Fatalf("expected healthy meta after check: %+v", checkedMeta)
	}
	if len(checkedMeta.Locators) != 1 {
		t.Fatalf("expected single backup locator, got %+v", checkedMeta.Locators)
	}

	restoreDir := filepath.Join(tmpDir, "restore-101")
	executeRoot(t, "--config", configPath, "ar", "unpack", "--cid", "101", "--out", restoreDir)
	assertFileContent(t, filepath.Join(restoreDir, "src-101", "a.txt"), "alpha")
	assertFileContent(t, filepath.Join(restoreDir, "src-101", "nested", "b.txt"), "bravo")
}

func TestArMongoPackQueryAndCheck(t *testing.T) {
	if os.Getenv("MONGO_TEST") == "" {
		t.Skip("MONGO_TEST not set")
	}

	tmpDir := t.TempDir()
	fake7z := writeFake7z(t, tmpDir)

	galleryRoot := filepath.Join(tmpDir, "gallery")
	archiveRoot := filepath.Join(tmpDir, "archive")
	archiveTempRoot := filepath.Join(tmpDir, "archive-temp")
	backupRoot := filepath.Join(tmpDir, "backup")
	backupBackend := "ar-backup-mongo"
	collectionName := fmt.Sprintf("comicInfo_ar_cli_%d", time.Now().UnixNano())
	configPath := writeArConfig(
		t,
		tmpDir,
		fake7z,
		galleryRoot,
		archiveRoot,
		archiveTempRoot,
		"",
		"",
		backupBackend,
		backupRoot,
		"mongo",
		collectionName,
	)

	loadConfigFile(t, configPath)
	db, err := mongowrap.DB(viper.GetString("comic.mongo.database"))
	if err != nil {
		t.Fatalf("db err: %v", err)
	}
	coll := db.Collection(collectionName)
	defer coll.Drop(context.Background())

	info := api.ComicInfo{CID: 901}
	info.Title.Pretty = "CLI Mongo"
	if _, err := coll.InsertOne(context.Background(), info); err != nil {
		t.Fatalf("seed comicInfo failed: %v", err)
	}
	writeFixtureFile(t, filepath.Join(info.SaveDir(), "page-1.txt"), "mongo-page")

	executeRoot(t, "--config", configPath, "ar", "pack", "--cid", "901")
	queryOutput := executeRoot(t, "--config", configPath, "ar", "--output", "json", "query", "--cid", "901")
	queryEnvelope := decodeEnvelope(t, queryOutput)
	queryMeta := decodeArchiveMeta(t, queryEnvelope["data"])
	if queryMeta.ID != 901 || queryMeta.Path == "" {
		t.Fatalf("unexpected mongo query result: %+v", queryMeta)
	}

	executeRoot(t, "--config", configPath, "ar", "check", "--cid", "901")

	var raw bson.M
	if err := coll.FindOne(context.Background(), bson.M{"cid": 901}).Decode(&raw); err != nil {
		t.Fatalf("load raw comicInfo failed: %v", err)
	}
	if title, ok := raw["title"].(bson.M); !ok || title["pretty"] != "CLI Mongo" {
		t.Fatalf("title subtree changed unexpectedly: %+v", raw)
	}
	archiveDoc, ok := raw["archive"].(bson.M)
	if !ok {
		t.Fatalf("archive subtree missing: %+v", raw)
	}
	if archiveDoc["path"] == "" || archiveDoc["algorithm"] == "" {
		t.Fatalf("archive compatibility fields missing: %+v", archiveDoc)
	}
	if _, ok := archiveDoc["manager"]; !ok {
		t.Fatalf("archive.manager missing: %+v", archiveDoc)
	}
}

func writeFake7z(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "fake7z.sh")
	content := "#!/bin/sh\n" +
		"set -eu\n" +
		"mode=\"$1\"\n" +
		"shift\n" +
		"if [ \"$mode\" = \"a\" ]; then\n" +
		"  dest=\"\"\n" +
		"  list=\"\"\n" +
		"  for arg in \"$@\"; do\n" +
		"    case \"$arg\" in\n" +
		"      -*) ;;\n" +
		"      @*) list=\"${arg#@}\" ;;\n" +
		"      *) if [ -z \"$dest\" ]; then dest=\"$arg\"; fi ;;\n" +
		"    esac\n" +
		"  done\n" +
		"  mkdir -p \"$(dirname \"$dest\")\"\n" +
		"  tar -cf \"$dest\" -T \"$list\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$mode\" = \"x\" ]; then\n" +
		"  out=\"\"\n" +
		"  src=\"\"\n" +
		"  for arg in \"$@\"; do\n" +
		"    case \"$arg\" in\n" +
		"      -o*) out=\"${arg#-o}\" ;;\n" +
		"      -*) ;;\n" +
		"      *) src=\"$arg\" ;;\n" +
		"    esac\n" +
		"  done\n" +
		"  mkdir -p \"$out\"\n" +
		"  tar -xf \"$src\" -C \"$out\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo unsupported mode: $mode >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake 7z failed: %v", err)
	}
	return path
}

func writeArConfig(
	t *testing.T,
	dir string,
	fake7z string,
	galleryRoot string,
	archiveRoot string,
	archiveTempRoot string,
	indexBackend string,
	indexRoot string,
	backupBackend string,
	backupRoot string,
	indexType string,
	collectionName string,
) string {
	t.Helper()

	var builder strings.Builder
	builder.WriteString("cocom:\n")
	builder.WriteString(fmt.Sprintf("  storage:\n    path: %q\n", galleryRoot))
	builder.WriteString("  archive:\n")
	builder.WriteString(fmt.Sprintf("    path: %q\n", archiveRoot))
	builder.WriteString(fmt.Sprintf("    temp_path: %q\n", archiveTempRoot))
	builder.WriteString(fmt.Sprintf("    password: %q\n", "test-password"))
	builder.WriteString(fmt.Sprintf("    cmd: %q\n", fake7z))
	builder.WriteString("    algorithm: single\n")
	builder.WriteString("archive:\n")
	builder.WriteString("  manager:\n")
	builder.WriteString("    algorithm: single\n")
	builder.WriteString("    replicates:\n")
	builder.WriteString("    index:\n")
	builder.WriteString(fmt.Sprintf("      type: %q\n", indexType))
	if indexType == "file" {
		builder.WriteString(fmt.Sprintf("      file_store_name: %q\n", indexBackend))
		builder.WriteString("      file_store_prefix: \"archive/index\"\n")
	}
	if indexType == "mongo" {
		builder.WriteString(fmt.Sprintf("      mongo_database: %q\n", "cocom"))
		builder.WriteString(fmt.Sprintf("      mongo_collection: %q\n", collectionName))
	}
	builder.WriteString("storage:\n")
	builder.WriteString("  backends:\n")
	if indexType == "file" {
		builder.WriteString(fmt.Sprintf("    - name: %q\n", indexBackend))
		builder.WriteString("      type: localfs\n")
		builder.WriteString("      metadata:\n")
		builder.WriteString(fmt.Sprintf("        root: %q\n", indexRoot))
	}
	builder.WriteString(fmt.Sprintf("    - name: %q\n", backupBackend))
	builder.WriteString("      type: localfs\n")
	builder.WriteString("      metadata:\n")
	builder.WriteString(fmt.Sprintf("        root: %q\n", backupRoot))
	builder.WriteString("mongo:\n")
	builder.WriteString("  database: cocom\n")
	builder.WriteString("comic:\n")
	builder.WriteString("  mongo:\n")
	if indexType == "mongo" {
		builder.WriteString("    database: cocom\n")
		builder.WriteString("    collections:\n")
		builder.WriteString(fmt.Sprintf("      comicInfo: %q\n", collectionName))
	}

	configPath := filepath.Join(dir, "cocom-ar.yaml")
	if err := os.WriteFile(configPath, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	return configPath
}

func executeRoot(t *testing.T, args ...string) string {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute %v failed: %v stderr=%s", args, err, stderr.String())
	}
	return stdout.String()
}

func decodeEnvelope(t *testing.T, output string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode json output failed: %v output=%s", err, output)
	}
	return payload
}

func decodeArchiveMeta(t *testing.T, raw any) manager.ArchiveMeta {
	t.Helper()
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal archive meta failed: %v", err)
	}
	var meta manager.ArchiveMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("unmarshal archive meta failed: %v", err)
	}
	return meta
}

func writeFixtureFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file failed: %v", err)
	}
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s failed: %v", path, err)
	}
	if string(content) != expected {
		t.Fatalf("unexpected content for %s: got=%q want=%q", path, string(content), expected)
	}
}

func loadConfigFile(t *testing.T, path string) {
	t.Helper()
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("load config failed: %v", err)
	}
}

func TestArchiveCLIEmitErrorJSON(t *testing.T) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	archivecli.EmitError(&stderr, &stdout, "json", fmt.Errorf("boom"))
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"ok\":false") {
		t.Fatalf("unexpected json error output: %s", stdout.String())
	}
}
