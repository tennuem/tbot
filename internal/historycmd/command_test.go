package historycmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/internal/history"
	"github.com/tennuem/tbot/pkg/service"
)

func TestHistoryPreviewApplyExportAndReimport(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	db := filepath.Join(dir, "cache.db")
	exported := filepath.Join(dir, "export.json")
	csv := filepath.Join(dir, "export.csv")
	archive := &history.Archive{Version: history.Version, Shares: []service.Share{{
		MessageID: 1, ChatID: -10042, UserID: 10, UserName: "First", SentAt: time.Now().UTC(),
		URL: "https://music.yandex.ru/album/1/track/2?utm_source=share",
	}}}
	data, err := json.Marshal(archive)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(input, data, 0600))
	var output bytes.Buffer
	require.NoError(t, Run(context.Background(), []string{"import", "--input", input, "--sqlite", db}, &output))
	require.Contains(t, output.String(), "Preview only")
	_, err = os.Stat(db)
	require.True(t, os.IsNotExist(err))
	output.Reset()
	require.NoError(t, Run(context.Background(), []string{"import", "--input", input, "--sqlite", db, "--apply"}, &output))
	require.Contains(t, output.String(), `"shares_added": 1`)
	require.NoError(t, Run(context.Background(), []string{"export", "--sqlite", db, "--output", exported, "--csv", csv}, &output))
	file, err := os.Open(exported)
	require.NoError(t, err)
	got, err := history.Read(file)
	require.NoError(t, file.Close())
	require.NoError(t, err)
	require.Len(t, got.Shares, 1)
	require.Len(t, got.Cache, 1)
	require.Contains(t, got.Shares[0].URL, "music.yandex.com")
	info, err := os.Stat(exported)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	output.Reset()
	require.NoError(t, Run(context.Background(), []string{"import", "--input", exported, "--sqlite", db, "--apply"}, &output))
	require.Contains(t, output.String(), `"shares_added": 0`)
	require.Contains(t, output.String(), `"duplicate_shares": 1`)
	require.Error(t, Run(context.Background(), []string{"export", "--sqlite", db, "--output", exported}, &output))
	dataAfter, err := os.ReadFile(exported)
	require.NoError(t, err)
	require.NotEmpty(t, dataAfter)
}

func TestHistoryCLIRejectsInvalidInputsWithoutCreatingDatabase(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "missing.db")
	input := filepath.Join(dir, "invalid.json")
	require.NoError(t, os.WriteFile(input, []byte(`{"version":999}`), 0600))
	var out bytes.Buffer
	for _, args := range [][]string{
		{"import", "--input", input, "--sqlite", db, "--apply"},
		{"export", "--sqlite", db, "--output", filepath.Join(dir, "out.json")},
		{"extract"}, {"invalid-command"}, {"import"}, {"export"},
	} {
		require.Error(t, Run(context.Background(), args, &out), args)
	}
	_, err := os.Stat(db)
	require.True(t, os.IsNotExist(err))
	require.NoError(t, Run(context.Background(), nil, &out))
	require.NoError(t, Run(context.Background(), []string{"import", "--help"}, &out))
}

func TestHistoryExtractCLIPreservesDeletedAccountWithoutGuessingID(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "messages.html"), filepath.Join(dir, "history.json")
	text := `<title>Chat</title><div class="history"><div class="message default" id="message1"><div class="body">
		<div class="date" title="02.07.2019 12:25:35 UTC+03:00"></div><div class="from_name">Deleted Account</div>
		<div class="text">https://music.yandex.ru/album/1/track/2</div></div></div></div>`
	require.NoError(t, os.WriteFile(input, []byte(text), 0600))
	var out bytes.Buffer
	require.NoError(t, Run(context.Background(), []string{"extract", "--input", input, "--output", output, "--chat-id=-10042"}, &out))
	require.Contains(t, out.String(), `"unknown_authors": 1`)
	archive, err := readArchive(output)
	require.NoError(t, err)
	require.Zero(t, archive.Shares[0].UserID)
	require.Equal(t, "Deleted Account", archive.Shares[0].UserName)
	require.Equal(t, int64(-10042), archive.Shares[0].ChatID)
	require.Equal(t, 9, archive.Shares[0].SentAt.UTC().Hour())
}
