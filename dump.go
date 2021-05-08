package accounting_bot

import (
	"bytes"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"time"
)

var names = []string{"id", "created", "currency", "value", "comment", "tags"}

type csvDumper struct {
	entries []*Entry
	csv     *csv.Writer
	buff    *bytes.Buffer
	n       int
}

func (d *csvDumper) Read(p []byte) (int, error) {
	if d.n == 0 {
		if err := d.csv.Write(names); err != nil {
			return 0, err
		}
	}
	if d.n >= len(d.entries) {
		return 0, io.EOF
	}

	fields := []string{
		d.entries[d.n].ID,
		d.entries[d.n].CreatedAt.Format(time.RFC3339),
		d.entries[d.n].Currency,
		strconv.FormatFloat(float64(d.entries[d.n].Value), 'f', 4, 32),
		d.entries[d.n].Comment,
		strings.Join(d.entries[d.n].Tags, ","),
	}
	if err := d.csv.Write(fields); err != nil {
		return 0, err
	}
	d.csv.Flush()
	d.n++
	return d.buff.Read(p)
}

func (d *csvDumper) Close() error {
	return nil
}

func DumpCsv(entries []*Entry) (io.ReadCloser, error) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))
	w := csv.NewWriter(buff)
	return &csvDumper{entries: entries, csv: w, buff: buff}, nil
}
