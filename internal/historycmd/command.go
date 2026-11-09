// Package historycmd implements offline history tools in the tbot binary.
package historycmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tennuem/tbot/internal/history"
	"github.com/tennuem/tbot/internal/store/sqlite"
)

func Run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		fmt.Fprintln(out, "Usage: tbot history extract|import|export [options]")
		return nil
	}
	flags := flag.NewFlagSet("history "+args[0], flag.ContinueOnError)
	flags.SetOutput(out)
	input := flags.String("input", "", "Telegram HTML directory/file or JSON archive")
	output := flags.String("output", "", "new output JSON file (never overwritten)")
	csvPath := flags.String("csv", "", "optional new CSV file for review")
	timezone := flags.String("timezone", "Europe/Moscow", "timezone of Telegram HTML timestamps")
	chatID := flags.Int64("chat-id", 0, "bind unassigned shares to this Telegram chat ID")
	exclude := flags.String("exclude-users", "", "comma-separated user IDs to omit when extracting HTML")
	database := os.Getenv("TBOT_SQLITE_DATA_SOURCE")
	if database == "" {
		database = "./tbot.db"
	}
	sqlitePath := flags.String("sqlite", database, "SQLite file path")
	apply := flags.Bool("apply", false, "apply import; without this flag only validate and preview")
	allowIncomplete := flags.Bool("allow-incomplete", false, "allow HTML extraction with unknown authors or invalid dates")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	switch args[0] {
	case "extract":
		if *input == "" || *output == "" {
			return errors.New("extract requires --input and --output")
		}
		users, err := excludedUsers(*exclude)
		if err != nil {
			return err
		}
		archive, report, err := history.ExtractHTML(*input, history.ExtractOptions{
			ChatID: *chatID, Timezone: *timezone, ExcludeUsers: users,
		})
		if err != nil {
			return err
		}
		if !*allowIncomplete && (report.MissingAuthors != 0 || report.InvalidDates != 0 || report.InvalidLinks != 0) {
			return fmt.Errorf("incomplete HTML extraction: %d messages with unknown authors, %d invalid dates, %d invalid links; review export or use --allow-incomplete",
				report.MissingAuthors, report.InvalidDates, report.InvalidLinks)
		}
		if err := writeArchive(*output, *csvPath, archive); err != nil {
			return err
		}
		return printJSON(out, struct {
			history.ExtractReport
			history.Summary
		}{report, history.Summarize(archive)})
	case "import":
		archive, err := readArchive(*input)
		if err != nil {
			return err
		}
		archive.BindChat(*chatID)
		if err := archive.Validate(true); err != nil {
			return err
		}
		if !*apply {
			fmt.Fprintln(out, "Preview only: SQLite was not opened or modified. Add --apply to import.")
			return printJSON(out, history.Summarize(archive))
		}
		store, err := sqlite.NewSqLiteStore(*sqlitePath)
		if err != nil {
			return err
		}
		defer store.Close()
		result, err := store.ImportHistory(ctx, archive)
		if err != nil {
			return err
		}
		return printJSON(out, result)
	case "export":
		if *output == "" {
			return errors.New("export requires --output")
		}
		store, err := sqlite.OpenReadOnly(*sqlitePath)
		if err != nil {
			return err
		}
		defer store.Close()
		archive, err := store.ExportHistory(ctx)
		if err != nil {
			return err
		}
		if err := writeArchive(*output, *csvPath, archive); err != nil {
			return err
		}
		return printJSON(out, history.Summarize(archive))
	default:
		return errors.New("unknown history command; use extract, import, or export")
	}
}

func excludedUsers(raw string) (map[int]bool, error) {
	users := make(map[int]bool)
	if raw == "" {
		return users, nil
	}
	for _, text := range strings.Split(raw, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || id <= 0 {
			return nil, errors.New("--exclude-users must contain positive numeric IDs")
		}
		users[id] = true
	}
	return users, nil
}

func readArchive(path string) (*history.Archive, error) {
	if path == "" {
		return nil, errors.New("import requires --input")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return history.Read(file)
}

func printJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeArchive(path, csvPath string, archive *history.Archive) error {
	if csvPath == path {
		return errors.New("JSON and CSV output paths must differ")
	}
	if csvPath != "" {
		if _, err := os.Stat(csvPath); !os.IsNotExist(err) {
			return errors.New("CSV output already exists or is inaccessible")
		}
	}
	if err := writeNew(path, func(w io.Writer) error { return printJSON(w, archive) }); err != nil {
		return err
	}
	if csvPath != "" {
		return writeNew(csvPath, func(w io.Writer) error { return writeCSV(w, archive) })
	}
	return nil
}

func writeNew(path string, write func(io.Writer) error) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	err = write(file)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
	}
	return err
}

func writeCSV(out io.Writer, archive *history.Archive) error {
	zone := time.UTC
	if archive.Timezone != "" {
		var err error
		zone, err = time.LoadLocation(archive.Timezone)
		if err != nil {
			return err
		}
	}
	w := csv.NewWriter(out)
	if err := w.Write([]string{"sent_at", "user_id", "user_name", "chat_id", "message_id", "provider", "url", "title"}); err != nil {
		return err
	}
	for _, share := range archive.Shares {
		if err := w.Write([]string{share.SentAt.In(zone).Format(time.RFC3339), strconv.Itoa(share.UserID), csvText(share.UserName),
			strconv.FormatInt(share.ChatID, 10), strconv.Itoa(share.MessageID), share.Provider, share.URL, csvText(share.Title)}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func csvText(text string) string {
	if text != "" && strings.ContainsAny(text[:1], "=+-@\t\r") {
		return "'" + text
	}
	return text
}
