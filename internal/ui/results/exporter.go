package results

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"

	"github.com/sadopc/gotermsql/internal/adapter"
)

// ExportCSV writes the given columns and rows to a CSV file at path.
func ExportCSV(path string, columns []adapter.ColumnMeta, rows [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)

	// Write header row.
	header := make([]string, len(columns))
	for i, c := range columns {
		header[i] = c.Name
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// Write data rows.
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}

	w.Flush()
	return w.Error()
}

// ExportJSON writes the given columns and rows as a JSON array of objects
// to a file at path. Each object maps column names to string values.
func ExportJSON(path string, columns []adapter.ColumnMeta, rows [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	colNames := make([]string, len(columns))
	for i, c := range columns {
		colNames[i] = c.Name
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	// Build the full array so the output is a proper JSON array.
	// For in-memory exports the data fits in memory by definition.
	objects := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		obj := make(map[string]string, len(colNames))
		for j, name := range colNames {
			if j < len(row) {
				obj[name] = row[j]
			} else {
				obj[name] = ""
			}
		}
		objects = append(objects, obj)
	}

	return enc.Encode(objects)
}

// ExportCSVFromIterator streams rows from an adapter.RowIterator into a CSV
// file. It writes incrementally so that arbitrarily large result sets can be
// exported without holding all rows in memory. It returns the number of rows
// written.
func ExportCSVFromIterator(ctx context.Context, path string, iter adapter.RowIterator) (int64, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	w := csv.NewWriter(f)

	// Write header.
	cols := iter.Columns()
	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = c.Name
	}
	if err := w.Write(header); err != nil {
		return 0, err
	}

	var count int64
	for {
		if ctx.Err() != nil {
			w.Flush()
			return count, ctx.Err()
		}

		rows, err := iter.FetchNext(ctx)
		if err != nil {
			if adapter.SentinelEOF(err) {
				break
			}
			w.Flush()
			return count, err
		}

		for _, row := range rows {
			if writeErr := w.Write(row); writeErr != nil {
				w.Flush()
				return count, writeErr
			}
			count++
		}

		// Flush periodically to keep memory usage low.
		w.Flush()
		if flushErr := w.Error(); flushErr != nil {
			return count, flushErr
		}
	}

	w.Flush()
	return count, w.Error()
}

// ExportJSONFromIterator streams rows from an adapter.RowIterator into a
// JSON file as an array of objects. It writes incrementally, flushing each
// page to disk so that large datasets do not require holding all data in
// memory. It returns the number of rows written.
func ExportJSONFromIterator(ctx context.Context, path string, iter adapter.RowIterator) (int64, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	cols := iter.Columns()
	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.Name
	}

	// Write opening bracket.
	if _, err := io.WriteString(f, "[\n"); err != nil {
		return 0, err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("  ", "  ")

	var count int64
	for {
		if ctx.Err() != nil {
			// Write closing bracket even on cancellation for partial validity.
			io.WriteString(f, "\n]") //nolint:errcheck
			return count, ctx.Err()
		}

		rows, err := iter.FetchNext(ctx)
		if err != nil {
			if adapter.SentinelEOF(err) {
				break
			}
			io.WriteString(f, "\n]") //nolint:errcheck
			return count, err
		}

		for _, row := range rows {
			obj := make(map[string]string, len(colNames))
			for j, name := range colNames {
				if j < len(row) {
					obj[name] = row[j]
				} else {
					obj[name] = ""
				}
			}

			// Write comma separator between objects.
			if count > 0 {
				if _, err := io.WriteString(f, ",\n"); err != nil {
					return count, err
				}
			} else {
				if _, err := io.WriteString(f, "  "); err != nil {
					return count, err
				}
			}

			data, err := json.MarshalIndent(obj, "  ", "  ")
			if err != nil {
				return count, err
			}
			if _, err := f.Write(data); err != nil {
				return count, err
			}

			count++
		}
	}

	// Write closing bracket.
	if _, err := io.WriteString(f, "\n]\n"); err != nil {
		return count, err
	}

	return count, nil
}
